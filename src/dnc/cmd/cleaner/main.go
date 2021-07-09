package main

import (
	"context"
	ad "dnc/activedirectory"
	"dnc/configuration"
	"dnc/datamodel"
	"dnc/datastore/azurestorage"
	"dnc/datastoreshim"
	"dnc/dns"
	"dnc/logger/loggershim"
	"dnc/logger/simplejson"
	"dnc/pubsubproxy"
	"dnc/utils"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"time"

	"github.com/Azure/azure-sdk-for-go/storage"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	metrics "github.com/slok/go-http-metrics/metrics/prometheus"
	"github.com/slok/go-http-metrics/middleware"
	"github.com/slok/go-http-metrics/middleware/std"
)

type partitionRepo struct {
	dataStoreShim              datastoreshim.DataStoreShim
	ncRepo                     *datamodel.NetworkContainerRepo
	reservationRepo            *datamodel.ReservationRepo
	ipAllocationRepo           *datamodel.AllocationRepo
	pendingPublishRequestRepo  *datamodel.NcPendingPublishRequestRepo
	pendingResourceCleanupRepo *datamodel.PendingResourceCleanupRepo
	nodeRepo                   *datamodel.NodeConfigurationRepo
	MaxBatchSize               int
}

func Log(ctx context.Context) loggershim.LoggerInterface {
	logger, ok := ctx.Value("logger").(loggershim.LoggerInterface)
	if ok {
		return logger
	}
	return loggershim.LS
}

// HTTP API to process /clean/partition API
// This will cleanup all the entries for a given DNC partition and unpublish the associated
// state from pubsub
func CleanPartition(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	partition, ok := vars["partition"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ctx := context.WithValue(r.Context(), "logger", jsonlogger.WithField("partition", partition))
	Log(ctx).Debugf("Cleaning")

	// Initialize AzStorage for this partition
	ds, err := datastoreshim.NewDataStoreShimEx(
		partition,
		config.DataStoreSettings.AzureStorageInfo.IsUsingCosmosDb,
		azStorageClient,
		config.DNCSettings.ThresholdForLoggingDataStoreOpLatencyMs,
		datamodel.TableToColumnMap,
		config.DataStoreSettings.MaxBatchSize)

	if err != nil {
		Log(ctx).Errorf("could not initialize azure storage: %v", err)
		json.NewEncoder(w).Encode(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	partitionRepo := &partitionRepo{
		dataStoreShim:              ds,
		ncRepo:                     datamodel.NewNetworkContainerRepo(ds, partition, Log(ctx)),
		reservationRepo:            datamodel.NewReservationRepo(ds, partition, Log(ctx)),
		ipAllocationRepo:           &datamodel.AllocationRepo{Datastore: ds},
		pendingPublishRequestRepo:  datamodel.NewNcPendingPublishRequestRepo(ds, partition),
		pendingResourceCleanupRepo: datamodel.NewPendingResourceCleanupRepo(ds, Log(ctx)),
		nodeRepo:                   &datamodel.NodeConfigurationRepo{Datastore: ds},
		MaxBatchSize:               config.DataStoreSettings.MaxBatchSize,
	}

	// check if work is still pending to unpublish the state and release ips
	workRequired, err := IsCleanupRequired(ctx, partitionRepo)
	if err != nil {
		Log(ctx).Errorf("Issue checking cleanup %s: %s", partition, err)
		json.NewEncoder(w).Encode(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	Log(ctx).Debugf("Work required for Partition: %s: %t", partition, workRequired)

	if workRequired {
		// Work is required to unpublish and/or cleanup associated ips
		// Do it asynchronously
		go func() {
			// todo: Unpublish and then release the associated ips
			err := CleanupPartitionState(ctx, partitionRepo)
			if err != nil {
				Log(ctx).Errorf("Issue with deleting unpublish or ipallocation state: %s", err)
				return
			}

			Log(ctx).Debugf("Now we can safely wipe all the tables")
			//we could just let the next call handle this
			err = wipeAllAssociatedData(ctx, partitionRepo)
			if err != nil {
				Log(ctx).Errorf("Issue with wiping %s: %s", partition, err)
				return
			}
			Log(ctx).Debugf("Cleanup done for %s", partition)
		}()

		w.WriteHeader(http.StatusAccepted)
		return
	}

	err = wipeAllAssociatedData(ctx, partitionRepo)
	if err != nil {
		json.NewEncoder(w).Encode(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	Log(ctx).Debugf("Cleanup done for %s", partition)
	// Safe to return 404, caller will treat this as success
	w.WriteHeader(http.StatusNotFound)
}

// Check the key tables if data unpublish or IPAllocations cleanup is pending
func IsCleanupRequired(ctx context.Context, repo *partitionRepo) (bool, error) {
	// Check if this partition has some NC data which requires to be unpublished from PubSub
	cleanupRequired, err := repo.ncRepo.AnyNetworkContainerExists()
	if err != nil {
		return false, err
	}

	if cleanupRequired {
		Log(ctx).Debugf("IsCleanupRequired: Pending NetworkContainers exists")
		return true, nil
	}

	cleanupRequired, err = repo.pendingPublishRequestRepo.AnyPendingPublishRequestExists()
	if err != nil {
		return false, err
	}

	if cleanupRequired {
		Log(ctx).Debugf("IsCleanupRequired partition: Pending PublishRequests exists")
		return true, nil
	}

	// This means all data is unpublished, now check if IP cleanup is pending
	// Check if PubSub has any pending rows
	cleanupRequired, err = repo.pendingResourceCleanupRepo.AnyPendingResourceCleanupExists()
	if err != nil {
		return false, err
	}

	if cleanupRequired {
		Log(ctx).Debugf("IsCleanupRequired partition: Pending ResourceCleanup exists")
		return true, nil
	}

	// All Pending Resource cleanup is gone, check if any pending reservations entry
	// Todo: do we need this as it must exist if NCs entry exist
	cleanupRequired, err = repo.reservationRepo.AnyReservationConfigurationExists()
	if err != nil {
		return false, err
	}

	if cleanupRequired {
		Log(ctx).Debugf("IsCleanupRequired partition %s: Pending Reservations exists")
		return true, nil
	}

	return cleanupRequired, nil
}

// This will unpublish the pending NC state and then delete the associated ips
func CleanupPartitionState(ctx context.Context, repo *partitionRepo) error {
	err := UnpublishNCState(ctx, repo)
	if err != nil {
		Log(ctx).Errorf("Failed to unpublish the NC goal state, err %+v", err)
		return err
	}

	err = DeleteAllocatedIpsForPartition(ctx, repo)
	if err != nil {
		Log(ctx).Errorf("Failed to delete associated allocatedIps, err %+v", err)
		return err
	}
	return nil
}

func wipeAllAssociatedData(ctx context.Context, repo *partitionRepo) error {
	// All the bg cleanup work is done, as a safe measure wipe all the tables again for this partition
	// Skip IpAllocations table as that is not partitioned per Partition Key and has already been taken
	// care
	for tableName, _ := range datamodel.TableToColumnMap {
		if tableName != datamodel.IPAllocationsTableName {
			err := repo.dataStoreShim.WipePartition(tableName)
			if err != nil {
				Log(ctx).Errorf("Failed to wipe table entities for tableName %s, err %v", tableName, err)
				//write the err to body
				return err
			}
		}
	}

	return nil
}

var (
	jsonlogger      simplejson.ExpandableLogger = simplejson.NewLogger("DNCCleanupEvent").WithField("Version", utils.Version)
	azStorageClient storage.Client
	config          *configuration.Configuration
)

//mirrors https://msazure.visualstudio.com/One/_git/Networking-Aquarius?path=%2Fsrc%2Fdnc%2Fserver%2Fhealthz.go&version=GC5bd8c32d2e83e13ffb3a5844f5dcd61b761fbb7b
//but without
func Health(w http.ResponseWriter, r *http.Request) {
	// Initialize AzStorage for this partition
	healthlogger := jsonlogger.WithField("healthprobe", r.URL.Path)
	ds, err := datastoreshim.NewDataStoreShimEx(
		"dummy",
		config.DataStoreSettings.AzureStorageInfo.IsUsingCosmosDb,
		azStorageClient,
		config.DNCSettings.ThresholdForLoggingDataStoreOpLatencyMs,
		datamodel.TableToColumnMap,
		config.DataStoreSettings.MaxBatchSize)

	if err != nil {
		healthlogger.Errorf("[health probe]  could not initialize azure storage: %v", err)
		json.NewEncoder(w).Encode(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	entity, _, err := ds.QueryInPartition(datamodel.PendingResourceCleanupTableName, "dummy", "dummy")
	if entity != nil {
		healthlogger.Debugf("[health probe] unexpected row found: %+v", *entity)
		w.WriteHeader(http.StatusOK)
		return
	}

	stErr, ok := err.(storage.AzureStorageServiceError)
	if !ok || stErr.StatusCode != http.StatusNotFound {
		json.NewEncoder(w).Encode(err)
		healthlogger.Errorf("[health probe] failed to query cosmos db %s: %s", config.DataStoreSettings.AzureStorageInfo.CosmosDBResourceID, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func main() {
	configpath, found := os.LookupEnv("DNCCLEANUP_CONFIGURATION_PATH")
	if !found {
		// this will be same as DNC config file as many of the modules are shared
		configpath = "config.json"
	}
	config = configuration.Load(configpath)
	configuration.OverrideConfigFromEnv(config)
	configuration.SetDncConfigDefaults(config)

	loggershim.LS = jsonlogger

	tokenFetcher := &ad.MSITokenFetcher{ResourceID: config.IdentitySettings.MSIResourceID}
	pubSubCfg := config.PubSubProxySettings
	ctx := context.Background()
	env, err := azure.EnvironmentFromName(config.AzureEnvironment.EnvironmentName)
	if err != nil {
		log.Fatalf("Invalid azure environment %s", config.AzureEnvironment.EnvironmentName)
	}
	if pubSubCfg.EnablePubSubProxy {
		// if a PubsubVipsString is not set, do a DNS lookup for PubsubVips instead
		if pubSubCfg.PubsubVipsString == "" && pubSubCfg.PubsubConnectionEndpoint != "" {
			log.Println("No PubsubVipsString given, using DNS to obtain PubSub VIPs.")
			translatedEndpoint := dns.TranslateEndpoint(pubSubCfg.PubsubConnectionEndpoint)
			pubsubvips, err := dns.PubsubVipsFromDNS(ctx, translatedEndpoint)
			if err != nil {
				log.Fatalf("Error obtaining PubsubVips from DNS using translated endpoint: %s, Error: %v", translatedEndpoint, err)
			}
			// TODO: allow overriding of the number of these based on a new config value (ie. PubsubVipsCount)
			//       then cut off any extra we received from DNS
			pubSubCfg.PubsubConnectionEndpoint = translatedEndpoint
			pubSubCfg.PubsubVips = pubsubvips
		}

		akv, err := ad.NewKeyvaultClient(tokenFetcher, env, pubSubCfg.PubsubCertKeyvaultName)
		if err != nil {
			log.Fatalln("Error initializing keyvault client. Error:", err)
		}

		certPath, err := getCertPath(ctx, akv, pubSubCfg.PubsubCertName)
		if err != nil {
			log.Fatalln("Error writing temporary cert file. Error:", err)
		}

		//be nice if this returned a pubsub we could pass to the server.
		if err := pubsubproxy.InitCleanup(pubSubCfg, certPath, loggershim.LS); err != nil {
			log.Fatalln("Error initializing PubsubProxy. Error:", err)
		}
	}

	if config.DataStoreSettings.AzureStorageInfo.CosmosDBResourceID == "" {
		log.Fatalln("cosmos db id reqired")
	}

	spt, err := tokenFetcher.GetServicePrincipalToken(env.ResourceManagerEndpoint)
	if err != nil {
		log.Fatalln("Error getting service principal token. Error:", err)
	}

	token, err := ad.GetFreshToken(ctx, spt)
	if err != nil {
		log.Fatalln("Error getting ARM token. Error:", err)
	}

	// todo: discuss with team: what should be our workflow for key rotation?
	keyFetcher := ad.CosmosKeyFetcher{ARMEndpoint: env.ResourceManagerEndpoint}
	keys, err := keyFetcher.GetReadWriteKeys(ctx, token, config.DataStoreSettings.AzureStorageInfo.CosmosDBResourceID)
	if err != nil {
		log.Fatalln("Error getting Cosmos RW Keys. Error:", err)
	}

	config.DataStoreSettings.AzureStorageInfo.Key = keys.PrimaryMasterKey

	loggershim.LS.Debugf("autoscaling tables")

	// initialize Storage Client
	configStorageInfo := config.DataStoreSettings.AzureStorageInfo
	azStorageClient, err = utils.GetStorageClient(
		configStorageInfo.AccountName,
		configStorageInfo.Key,
		configStorageInfo.Environment,
		configStorageInfo.IsUsingCosmosDb)
	if err != nil {
		log.Fatalf("Create storage client failed: %v", err)
	}

	azStorage, err := azurestorage.NewAzureStorageEx("dummy", true, azStorageClient, loggershim.LS, datamodel.TableToColumnMap,
		config.DNCSettings.ThresholdForLoggingDataStoreOpLatencyMs, config.DataStoreSettings.MaxBatchSize)
	if err != nil {
		log.Fatalf("could not initialize azure storage: %v", err)
	}

	err = azStorage.EnsureAutoScaledTables(ctx, config.DataStoreSettings.AzureStorageInfo.CosmosDBResourceID, token,
		env.ResourceManagerEndpoint, config.DataStoreSettings.AzureStorageInfo.AutoScaleMaxThroughput)
	if err != nil {
		log.Fatalln("Error Autoscaling Tables Error:", err)
	}

	router := mux.NewRouter().StrictSlash(true)

	// Create our middleware.
	mdlw := middleware.New(middleware.Config{
		Service:  "dnc-cleanup-" + utils.Version,
		Recorder: metrics.NewRecorder(metrics.Config{}),
	})

	//the string used in in clean partition here keeps the handler dimension from exploding by ignoring the parition in metrics
	//https://github.com/slok/go-http-metrics/#custom-handler-id
	cleanhandler := std.Handler("/clean/:partition", mdlw, http.HandlerFunc(CleanPartition))
	healthz := std.Handler("", mdlw, http.HandlerFunc(Health))
	router.Methods(http.MethodDelete).
		Path("/clean/{partition}").
		Handler(cleanhandler)
	router.Methods(http.MethodGet).
		Path("/healthz").
		Handler(healthz)

	// Serve our metrics.
	go func() {
		loggershim.LS.Debugf("metrics listening at 8081")
		if err := http.ListenAndServe(":8081", promhttp.Handler()); err != nil {
			log.Panicf("error while serving metrics: %s", err)
		}
	}()

	loggershim.LS.Debugf("Serving on port 9000")
	r := handlers.LoggingHandler(jsonlogger, panicHandler{router})
	http.ListenAndServe(":9000", r)
}

type panicHandler struct {
	handler http.Handler
}

func (p panicHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			stack := string(debug.Stack())
			loggershim.LS.Errorf("Panic for request %s: %s", req.URL, stack)
		}
	}()

	p.handler.ServeHTTP(w, req)
}

// getCertPath fetches a certificate from keyvault, writes it to a temporary file, and returns its file path
func getCertPath(ctx context.Context, kvc *ad.KeyvaultClient, certName string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	bs, err := kvc.GetLatestSecretPEM(ctx, certName)
	if err != nil {
		return "", fmt.Errorf("failed to fetch keyvault certificate: %v", err)
	}

	return utils.ToTempFile("pubsubcert-*.pem", bs)
}
