package cnms

import (
	"fmt"
	"strings"

	"github.com/Azure/azure-container-networking/ebtables"
	"github.com/Azure/azure-container-networking/log"
	"github.com/Azure/azure-container-networking/platform"
)

func (networkMonitor *NetworkMonitor) deleteRulesNotExistInMap(chainRules map[string]string, stateRules map[string]string) {
	for rule, chain := range chainRules {
		if _, ok := stateRules[rule]; !ok {
			if itr, ok := networkMonitor.DeleteRulesToBeValidated[rule]; ok && itr > 0 {
				log.Printf("Deleting Ebtable rule as it didn't exist in state for %d iterations chain %v rule %v", itr, chain, rule)
				if err := ebtables.DeleteEbtableRule(chain, rule); err != nil {
					log.Printf("Error while deleting ebtable rule %v", err)
				}

				delete(networkMonitor.DeleteRulesToBeValidated, rule)
			} else {
				log.Printf("[DELETE] Found unmatched rule chain %v rule %v itr %d. Giving one more iteration", chain, rule, itr)
				networkMonitor.DeleteRulesToBeValidated[rule] = itr + 1
			}
		}
	}
}

func deleteRulesExistInMap(originalChainRules map[string]string, stateRules map[string]string) {
	for rule, chain := range originalChainRules {
		if _, ok := stateRules[rule]; ok {
			if err := ebtables.DeleteEbtableRule(chain, rule); err != nil {
				log.Printf("Error while deleting ebtable rule %v", err)
			}
		}
	}
}

func (networkMonitor *NetworkMonitor) addRulesNotExistInMap(
	stateRules map[string]string,
	chainRules map[string]string) {

	for rule, chain := range stateRules {
		if _, ok := chainRules[rule]; !ok {
			if itr, ok := networkMonitor.AddRulesToBeValidated[rule]; ok && itr > 0 {
				log.Printf("Adding Ebtable rule as it didn't exist in state for %d iterations chain %v rule %v", itr, chain, rule)
				if err := ebtables.AddEbtableRule(chain, rule); err != nil {
					log.Printf("Error while adding ebtable rule %v", err)
				}

				delete(networkMonitor.AddRulesToBeValidated, rule)
			} else {
				log.Printf("[ADD] Found unmatched rule chain %v rule %v itr %d. Giving one more iteration", chain, rule, itr)
				networkMonitor.AddRulesToBeValidated[rule] = itr + 1
			}
		}
	}
}

func (networkMonitor *NetworkMonitor) CreateRequiredL2Rules(
	currentEbtableRulesMap map[string]string,
	currentStateRulesMap map[string]string) error {

	for rule := range networkMonitor.AddRulesToBeValidated {
		if _, ok := currentStateRulesMap[rule]; !ok {
			delete(networkMonitor.AddRulesToBeValidated, rule)
		}
	}

	networkMonitor.addRulesNotExistInMap(currentStateRulesMap, currentEbtableRulesMap)
	//TODO: call insertRuleToForwardToAzureChain

	return nil
}

func (networkMonitor *NetworkMonitor) RemoveInvalidL2Rules(
	currentEbtableRulesMap map[string]string,
	currentStateRulesMap map[string]string) error {

	for rule := range networkMonitor.DeleteRulesToBeValidated {
		if _, ok := currentEbtableRulesMap[rule]; !ok {
			delete(networkMonitor.DeleteRulesToBeValidated, rule)
		}
	}

	if networkMonitor.IsMonitorAllChain {
		originalChainRules := make(map[string]string)

		if err := generateL2RulesMap(originalChainRules, ebtables.PreRouting); err != nil {
			return err
		}

		if err := generateL2RulesMap(originalChainRules, ebtables.PostRouting); err != nil {
			return err
		}

		deleteRulesExistInMap(originalChainRules, currentEbtableRulesMap)
	}

	networkMonitor.deleteRulesNotExistInMap(currentEbtableRulesMap, currentStateRulesMap)
	return nil
}

func generateL2RulesMap(currentEbtableRulesMap map[string]string, chainName string) error {
	cmd := fmt.Sprintf("ebtables -t nat -L %v --Lmac2", chainName)
	rules, err := platform.ExecuteCommand(cmd)
	if err != nil {
		log.Printf("Error while getting rules list %v for chain %v", err, chainName)
		return err
	}

	rulesList := strings.Split(rules, "\n")
	log.Printf("Rules count : %v", len(rulesList)-4)

	for _, rule := range rulesList {
		rule = strings.TrimSpace(rule)
		if rule != "" && !strings.Contains(rule, "Bridge table") && !strings.Contains(rule, "Bridge chain") {
			currentEbtableRulesMap[rule] = chainName
		}
	}

	return nil
}

func GetEbTableRulesInMap() (map[string]string, error) {
	currentEbtableRulesMap := make(map[string]string)

	if err := generateL2RulesMap(currentEbtableRulesMap, ebtables.AzurePreRouting); err != nil {
		return nil, err
	}

	if err := generateL2RulesMap(currentEbtableRulesMap, ebtables.AzurePostRouting); err != nil {
		return nil, err
	}

	return currentEbtableRulesMap, nil
}
