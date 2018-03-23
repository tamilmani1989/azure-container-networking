package azuresql

import (
	"context"
	"database/sql"
	"dnc/configuration"
	"fmt"
	"log"
	"net/http"
	"reflect"

	"github.com/denisenkom/go-mssqldb"
)

// AzureSQL = Storage object to manage tables in a partition
type AzureSQL struct {
	configuration.AzureSQLInfo
	db               *sql.DB
	tableToColumnMap map[string]string
}

const (
	rowKeyString              = "RowKey"
	tableExistsErrorFmtString = "There is already an object named '%s' in the database."
	updateCmdFmtString        = "UPDATE %s SET %s = @%s WHERE %s = @%s"
	insertCmdFmtString        = "INSERT INTO %s (%s, %s) Values (@%s, @%s);"
	selectWhereCmdFmtString   = "SELECT %s FROM %s WHERE %s = @%s;"
	selectFmtString           = "SELECT %s, %s FROM %s;"
	getCountFmtString         = "SELECT COUNT(*) as count FROM %s WHERE %s = @%s;"
	deleteFmtString           = "DELETE FROM %s WHERE %s = @%s;"
)

func isTransientSQLError(errorCode int32) bool {
	switch errorCode {
	// SQL Error Code: 40501
	// The service is currently busy. Retry the request after 10 seconds. Code: (reason code to be decoded).
	case 40501:
		return true

	// SQL Error Code: 10928
	// Resource ID: %d. The %s limit for the database is %d and has been reached.
	case 10928:
		return true

		// SQL Error Code: 10929
		// Resource ID: %d. The %s minimum guarantee is %d, maximum limit is %d and the current usage for the database is %d.
		// However, the server is currently too busy to support requests greater than %d for this database.
	case 10929:
		return true

	// SQL Error Code: 10053
	// A transport-level error has occurred when receiving results from the server.
	// An established connection was aborted by the software in your host machine.
	case 10053:
		return true

	// SQL Error Code: 10054
	// A transport-level error has occurred when sending the request to the server.
	// (provider: TCP Provider, error: 0 - An existing connection was forcibly closed by the remote host.)
	case 10054:
		return true

	// SQL Error Code: 10060
	// A network-related or instance-specific error occurred while establishing a connection to SQL Server.
	// The server was not found or was not accessible. Verify that the instance name is correct and that SQL Server
	// is configured to allow remote connections. (provider: TCP Provider, error: 0 - A connection attempt failed
	// because the connected party did not properly respond after a period of time, or established connection failed
	// because connected host has failed to respond.)"}
	case 10060:
		return true

	// SQL Error Code: 18401
	// Login failed for user '%s'. Reason: Server is in script upgrade mode. Only administrator can connect at this time.
	// Devnote: this can happen when SQL is going through recovery (e.g. after failover)
	case 18401:
		return true

	// SQL Error Code: 40197
	// The service has encountered an error processing your request. Please try again.
	case 40197:
		return true

	// SQL Error Code: 40540
	// The service has encountered an error processing your request. Please try again.
	case 40540:
		return true

	// SQL Error Code: 40613
	// Database XXXX on server YYYY is not currently available. Please retry the connection later. If the problem persists, contact customer
	// support, and provide them the session tracing ID of ZZZZZ.
	case 40613:
		return true

	// SQL Error Code: 40143
	// The service has encountered an error processing your request. Please try again.
	case 40143:
		return true

	// SQL Error Code: 233
	// The client was unable to establish a connection because of an error during connection initialization process before login.
	// Possible causes include the following: the client tried to connect to an unsupported version of SQL Server; the server was too busy
	// to accept new connections; or there was a resource limitation (insufficient memory or maximum allowed connections) on the server.
	// (provider: TCP Provider, error: 0 - An existing connection was forcibly closed by the remote host.)
	case 233:
		return true

	// SQL Error Code: 64
	// A connection was successfully established with the server, but then an error occurred during the login process.
	// (provider: TCP Provider, error: 0 - The specified network name is no longer available.)
	case 64:
		return true

	}

	return false

}

func isSQLObjectExistError(err error) bool {
	if driverErr, ok := err.(mssql.Error); ok { // Now the error number is accessible directly
		if driverErr.Number == 2714 {
			return true
		}
	}
	return false
}

func getHTTPStatusCode(err error) int {
	fmt.Println("Error Type:", reflect.TypeOf(err))
	if driverErr, ok := err.(mssql.Error); ok { // Now the error number is accessible directly
		isTransient := isTransientSQLError(driverErr.Number)
		if isTransient {
			return http.StatusInternalServerError
		}
		return http.StatusUnprocessableEntity
	}

	return http.StatusInternalServerError
}

// Init - Initialize azure storage.
func (azureSQL *AzureSQL) Init(tableToColumnMap map[string]string) {
	var err error
	var tx *sql.Tx
	var connectionString = fmt.Sprintf("server=%s;user id=%s;password=%s;port=%d;database=%s;",
		azureSQL.AzureSQLInfo.Host,
		azureSQL.AzureSQLInfo.User,
		azureSQL.AzureSQLInfo.Password,
		azureSQL.AzureSQLInfo.Port,
		azureSQL.AzureSQLInfo.Database)
	azureSQL.tableToColumnMap = tableToColumnMap
	azureSQL.db, err = sql.Open("sqlserver", connectionString)
	err = azureSQL.Ping()
	if err != nil {
		log.Fatalf("Failed to connect to database due to error %s", err.Error())
	}

	tx, err = azureSQL.db.Begin()
	if err != nil {
		log.Fatalf("Failed to being transaction due to error :%s", err.Error())
	}

	defer func() {
		if err != nil {
			fmt.Println("Rolling back transaction due to error", err.Error())
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	for tableName, columnName := range tableToColumnMap {
		createTableCmd := fmt.Sprintf(`CREATE TABLE %s
			(%s varchar(255) PRIMARY KEY NOT NULL,
			%s text NULL)`,
			tableName,
			rowKeyString,
			columnName)
		_, err = tx.Exec(createTableCmd)
		if err != nil {
			if !isSQLObjectExistError(err) {
				log.Fatalf("Create table failed %s due to error %s", createTableCmd, err.Error())
			}
			err = nil
		}
	}
}

// Delete - Deletes a row in a given table.
func (azureSQL *AzureSQL) Delete(tableName string, rowKey string) error {
	var err error
	err = azureSQL.Ping()
	if err != nil {
		return err
	}

	tx, err := azureSQL.db.Begin()
	if err != nil {
		log.Printf("Failed to being transaction due to error :%s", err.Error())
		return err
	}

	defer func() {
		if err != nil {
			fmt.Println("Rolling back transaction due to error", err.Error())
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	deleteCmd := fmt.Sprintf(deleteFmtString, tableName, rowKeyString, rowKeyString)
	_, err = tx.Exec(
		deleteCmd,
		sql.Named(rowKeyString, rowKey))

	if err != nil {
		log.Panicf("Error deleting row Delete Cmd:%s, Error:%s", deleteCmd, err.Error())
		return err
	}

	return nil
}

// Ping - Ping to check if DB is alive.
func (azureSQL *AzureSQL) Ping() error {
	ctx := context.Background()
	err := azureSQL.db.PingContext(ctx)
	if err != nil {
		log.Panicf("Error pinging database: " + err.Error())
	}

	return err
}

// InsertOrUpdate - InsertOrUpdate a row in a given table.
func (azureSQL *AzureSQL) InsertOrUpdate(tableName, rowKey string, propertyName string, value interface{}) error {
	return azureSQL.insertOrUpdateRow(tableName, rowKey, propertyName, value)
}

// Query - Query a row in a given table.
func (azureSQL *AzureSQL) Query(tableName, rowKey string) (interface{}, int, error) {
	ctx := context.Background()

	// Check if database is alive.
	err := azureSQL.Ping()
	if err != nil {
		return nil, getHTTPStatusCode(err), err
	}

	selectCmd := fmt.Sprintf(selectWhereCmdFmtString,
		azureSQL.tableToColumnMap[tableName],
		tableName,
		rowKeyString,
		rowKeyString)

	// Execute query
	rows, err := azureSQL.db.QueryContext(
		ctx,
		selectCmd,
		sql.Named(rowKeyString, rowKey))
	if err != nil {
		log.Printf("Error reading rows. Cmd:%s, Error:%s", selectCmd, err.Error())
		return nil, -1, err
	}

	log.Printf("Cmd:%s", selectCmd)

	defer rows.Close()

	// Iterate through the result set.
	for rows.Next() {
		var data string

		// Get values from row.
		err := rows.Scan(&data)
		if err != nil {
			log.Printf("Error reading rows: " + err.Error())
			return nil, getHTTPStatusCode(err), err
		}

		return data, http.StatusOK, nil
	}

	return nil, http.StatusInternalServerError, err

}

// QueryAllRows - Query a row in a given table.
func (azureSQL *AzureSQL) QueryAllRows(tableName string, ctxt interface{}) (
	map[string]interface{}, bool, interface{}, int, error) {
	results := make(map[string]interface{})
	ctx := context.Background()

	// Check if database is alive.
	err := azureSQL.Ping()
	if err != nil {
		return nil, false, nil, getHTTPStatusCode(err), err
	}

	selectCmd := fmt.Sprintf(selectFmtString,
		rowKeyString,
		azureSQL.tableToColumnMap[tableName],
		tableName)

	// Execute query
	rows, err := azureSQL.db.QueryContext(
		ctx,
		selectCmd)
	if err != nil {
		log.Printf("Error reading rows, SelectCmd:%s, Error:%s", selectCmd, err.Error())
		return nil, false, nil, getHTTPStatusCode(err), err
	}

	defer rows.Close()

	// Iterate through the result set.
	for rows.Next() {
		var rowKey string
		var data string

		// Get values from row.
		err := rows.Scan(&rowKey, &data)
		if err != nil {
			log.Printf("Error parsing retrieved rows, SelectCmd:%s, Error:%s", selectCmd, err.Error())
			return nil, false, nil, getHTTPStatusCode(err), err
		}

		fmt.Printf("RowKey %s, Data %s", rowKey, data)
		results[rowKey] = data
	}

	return results, false, nil, http.StatusOK, nil
}

// KeyExist - check if key exist in a given table.
func (azureSQL *AzureSQL) KeyExist(tableName, rowKey string) (bool, error) {
	ctx := context.Background()

	// Check if database is alive.
	err := azureSQL.Ping()
	if err != nil {
		return false, err
	}

	countCmd := fmt.Sprintf(getCountFmtString, tableName, rowKeyString, rowKeyString)

	// Execute query
	rows, err := azureSQL.db.QueryContext(
		ctx,
		countCmd,
		sql.Named(rowKeyString, rowKey))
	if err != nil {
		log.Printf("Error reading rows, CountCmd:%s, Error:%s", countCmd, err.Error())
		return false, err
	}

	defer rows.Close()

	var count int
	for rows.Next() {
		err := rows.Scan(&count)
		fmt.Printf("Count%d", count)
		if err != nil {
			log.Printf("Failed to parse the retrieved count. CountCmd:%s, Error:%s", countCmd, err.Error())
			return false, err
		}
	}
	return count > 0, nil
}

func (azureSQL *AzureSQL) insertOrUpdateRow(tableName, rowKey, propertyName string, value interface{}) error {
	ctx := context.Background()
	var err error
	var result sql.Result
	var tx *sql.Tx

	err = azureSQL.Ping()
	if err != nil {
		return err
	}

	updateCmd := fmt.Sprintf(updateCmdFmtString, tableName, propertyName, propertyName, rowKeyString, rowKeyString)
	tx, err = azureSQL.db.Begin()
	if err != nil {
		log.Printf("Error when begining transaction. Error:%s", err.Error())
		return err
	}

	defer func() {
		if err != nil {
			fmt.Println("Rolling back due to error", err.Error())
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	result, err = tx.ExecContext(
		ctx,
		updateCmd,
		sql.Named(propertyName, value),
		sql.Named(rowKeyString, rowKey))

	if err != nil {
		log.Printf("Error updating row, UpdateCmd:%s, Error:%s", updateCmd, err.Error())
		return err
	}

	rowCount, err := result.RowsAffected()
	if err != nil {
		log.Printf("Failed to retrieve rows affected from the result, UpdateCmd:%s, Error:%s", updateCmd, err.Error())
		return err
	}

	if rowCount == 0 {
		insertCmd := fmt.Sprintf(insertCmdFmtString, tableName, rowKeyString, propertyName, rowKeyString, propertyName)
		result, err := tx.Exec(
			insertCmd,
			sql.Named(propertyName, value),
			sql.Named(rowKeyString, rowKey))
		if err != nil {
			log.Printf("Error inserting row, InsertCmd:%s, Error:%s", insertCmd, err.Error())
			return err
		}

		rowCount, err = result.RowsAffected()
		if err != nil {
			log.Printf("Failed to retrieve rows affected from the result, InsertCmd:%s, Error:%s", insertCmd, err.Error())
			return err
		}
		fmt.Println("Inserted RowCount", rowCount)
	}
	return err
}
