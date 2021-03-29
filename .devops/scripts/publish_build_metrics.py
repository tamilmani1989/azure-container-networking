import os
import sys
import requests
import json
import argparse
import datetime
import base64
from azure.storage import CloudStorageAccount
from utility import *

# Example usage python publish_build_metrics.py -definitionIds "9169" -publish console

def get_headers():
    personal_access_token = os.getenv("PAT")
    base_encoded_pat = base64.b64encode(bytes(f":{personal_access_token}", "utf-8")).decode("utf-8")
    return {
        "Authorization": f"Basic {base_encoded_pat}",
        "Content-Type": "application/json-patch+json"
    }

headers = get_headers()

def publish_range_build_metrics(min_finish_time_utc, max_finish_time_utc, build_definition_ids, publishing_target="console", org="msdata", project="project"):
    devops_base_url = f"https://dev.azure.com/{org}/{project}/_apis"
    for definition_id in build_definition_ids:
        try:
            builds = get_builds_range(devops_base_url, min_finish_time_utc, max_finish_time_utc, definition_id)
            for build in builds["value"]:
                publish_build_metrics(devops_base_url, build["id"], publishing_target)
        except:
            print(sys.exc_info()[0])
            continue


def get_builds_range(devops_base_url, min_finish_time_utc, max_finish_time_utc, definition_id):
    url = f"{devops_base_url}/build/builds?api-version=5.1&resultFilter=partiallySucceeded,succeeded&minTime={min_finish_time_utc}&maxTime={max_finish_time_utc}&definitions={definition_id}&queryOrder=finishTimeAscending"
    print(url)
    return requests.get(url, headers=headers).json()

def publish_build_metrics(devops_base_url, build_id, publishing_target):

    build = get_build(devops_base_url, build_id)

    # Get test run
    code_coverage = get_code_coverage(devops_base_url, build_id)
    metrics = extract_metrics(build, code_coverage)
    if(metrics is None):
        print(f"No code-coverage for build '{build_id}'")
        return

    if(publishing_target == "console"):
        pretty_print(metrics)
        return

    if(publishing_target == "storage"):
        publish_metrics_storage_account(metrics)

def publish_metrics_storage_account(metrics):
    account_name =  os.getenv("BUILD_TELEMETRY_STORAGE_ACCOUNT_NAME")
    account_key =  os.getenv("BUILD_TELEMETRY_STORAGE_ACCOUNT_KEY")
    table_name = 'qualitymetrics'
    print(f"Publishing to storage account '{account_name}' under table '{table_name}'.")
    pretty_print(metrics)

    account = CloudStorageAccount(account_name, account_key)
    table_service = account.create_table_service()
    try:
        table_service.insert_entity(table_name, metrics)
        print(f"Successfully inserted metrics in table '{table_name}'.")
    except:
        print(f"Failed to insert metrics in '{table_name}'.")
        print(sys.exc_info()[0], " occurred.")

def extract_metrics(build, code_coverage):
    metrics = {
        "definitionId" : build["definition"]["id"],
        "finishTimeUtc": build["finishTime"],
        'RowKey': str(build["id"]),
        'DefinitionName': build["definition"]["name"],
        'PartitionKey': str(build["project"]["name"]) + "-" + str(build["definition"]["id"]),
        'IsSynthetic': False
    }

    coverage_stats = get_line_code_coverage_stats(code_coverage)

    if(coverage_stats):
        metrics["totalLine"] = coverage_stats["total"]
        metrics["totalLineCovered"] = coverage_stats["covered"]
        metrics["coveragePercentage"] = coverage_stats["covered"] / coverage_stats["total"]
    else:
        return None
    return metrics

def get_line_code_coverage_stats(code_coverage):
    if not code_coverage["coverageData"]:
        return None
    
    for stat in code_coverage["coverageData"][0]["coverageStats"]:
        if(stat["label"].lower() == "lines"):
            return stat
    return None

def get_build(devops_base_url, build_id):
    url = f"{devops_base_url}/build/builds/{build_id}?api-version=5.1"
    print(url)
    return requests.get(url, headers=headers).json()

def get_code_coverage(devops_base_url, build_id):
    url = f"{devops_base_url}/test/codecoverage?buildId={build_id}&api-version=5.1-preview.1"
    return requests.get(url, headers=headers).json()

if __name__ == "__main__":
  parser = argparse.ArgumentParser(description='Publishes build metrics of a given DevOps build run.')
  parser.add_argument('-publish'      , type=str, help="Where to publish the metrics, either \"storage\" or \"console\".", default="console")
  parser.add_argument('-org'          , type=str, help="The devops organization, which can be interpolated from the devops host url. Ie {org}.visualstudio.com", default="msdata")
  parser.add_argument('-project'      , type=str, help="The name of the project where the build definition of the target build lies in", default="vienna")
  parser.add_argument('-minFinishTime', type=str, help="Filters builds that finished after this date.")
  parser.add_argument('-maxFinishTime', type=str, help="Filters builds that finished before this date.")
  parser.add_argument('-definitionIds' , type=str, nargs="*", help="The Azure DevOps build definition ID the build was queued for.", required=True)
  ns = parser.parse_args()
  target = ns.publish
  project = ns.project
  organization = ns.org
  max_finish_time_utc = ns.maxFinishTime
  min_finish_time_utc = ns.minFinishTime
  if not max_finish_time_utc:
      max_finish_time_utc = time_to_str(datetime.datetime.utcnow())
  if not min_finish_time_utc:
      min_finish_time_utc = time_to_str(datetime.datetime.utcnow() - datetime.timedelta(days=1))
  build_definition_ids = ns.definitionIds

  publish_range_build_metrics(min_finish_time_utc, max_finish_time_utc, build_definition_ids, target, organization, project)