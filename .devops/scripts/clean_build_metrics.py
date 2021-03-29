import os
import sys
import requests
import json
import argparse
from datetime import datetime, timedelta
import base64
from azure.storage import CloudStorageAccount
from utility import *
from pytz import timezone
import pytz
import time

account_name =  os.getenv("BUILD_TELEMETRY_STORAGE_ACCOUNT_NAME")
account_key =  os.getenv("BUILD_TELEMETRY_STORAGE_ACCOUNT_KEY")
Pipelines = os.getenv("Pipelines")

table_name = "qualitymetrics"

max_row_age = datetime.timedelta(days=240)
def daterange(start_date, end_date):
    date_range = (end_date - start_date).days
    for n in range(date_range):
        yield start_date + timedelta(n + 1)

def get_table_service():
    account = CloudStorageAccount(account_name, account_key)
    account.create_table_service()
    return account.create_table_service()

def get_cache_key(project):
    return project

def create_pipeline_set():
    output = {}
    if(Pipelines is None):
        return output
    
    covered_builds = Pipelines.split(" ")
    
    for definition_metadata in covered_builds:
        metadata = definition_metadata.split(":")
        project = metadata[1].lower()
        definitions = metadata[2].split("-")

        key = project
        output[key] = {}

        for definition in definitions:
            output[key][definition] = True
    return output

def is_row_expired(row, pipeline_cache):

    # handle deleted definitions
    if(Pipelines is not None):
        key = row["PartitionKey"].rsplit("-", 1)[0].lower()
        if(key not in pipeline_cache or str(row["definitionId"]) not in pipeline_cache[key]):
            return True
    return str_to_time(row["finishTimeUtc"]) < datetime.datetime.utcnow() - max_row_age

def get_partition_key_list(table_service):
    row_generator = table_service.query_entities(table_name, select="PartitionKey")
    unique_rows = {}
    for row in row_generator:
        unique_rows[row["PartitionKey"]] = {}
    
    return unique_rows.keys()

def get_rows_for_partition(table_service, partition_key):
    row_generator = table_service.query_entities(table_name, filter=f"PartitionKey eq '{partition_key}'")
    rows = []
    for row in row_generator:
        rows.append(row)
    return rows

def get_latest_metrics(current_date, min_date, row_cache, synthetic_row_cache):
    date_ptr = current_date
    # allow backtracking up to 7 days prior to get latest code coverage
    min_date -= timedelta(7)
    while(date_ptr > min_date):
        date_ptr -= timedelta(1)
        if(date_ptr in row_cache.keys()):
            return row_cache[date_ptr]
    date_ptr = current_date
    while(date_ptr > min_date):
        date_ptr -= timedelta(1)
        if(date_ptr in synthetic_row_cache.keys()):
            return synthetic_row_cache[date_ptr]
    return None

def create_metric(row, finish_time_utc):
    return {
        "definitionId" : row["definitionId"],
        "finishTimeUtc": time_to_str(finish_time_utc),
        "RowKey": row["RowKey"],
        "PartitionKey": row["PartitionKey"],
        "totalLine": row["totalLine"],
        "totalLineCovered": row["totalLineCovered"],
        "coveragePercentage": row["coveragePercentage"],
        'IsSynthetic': True
    }

def process_rows_from_partition(table_service, partition_key):
    row_cache = {}
    synthetic_row_cache = {}

    rows = get_rows_for_partition(table_service, partition_key)
    pipeline_cache = create_pipeline_set()
    for row in rows:
        # Delete expired rows
        if(is_row_expired(row, pipeline_cache)):
            print("Deleting " + str(row))
            table_service.delete_entity(table_name, row["PartitionKey"], row["RowKey"])
        else:
            finish_time_utc = str_to_time(row["finishTimeUtc"])
            finish_time_bucket = finish_time_utc.replace(microsecond=0, second= 0, minute=0, hour=0)

            # Add row to one of our cache
            if row["IsSynthetic"]:
                synthetic_row_cache[finish_time_bucket] = row
            else:
                row_cache[finish_time_bucket] = row

    #Fill in the gaps for nly the last 14 days. We're seeing issues with the storage account when nuking it with the whole dataset.
    fill_range = timedelta(days=60)
    max_date = datetime.datetime.utcnow().replace(microsecond=0, second= 0, minute=0, hour=0)
    min_date = max_date - fill_range
    
    # fill in the gaps
    for date in daterange(min_date, max_date):
        # Skip adding a synthetic marker if a build is already present for that day in either the real builds or synthetic builds
        if date in row_cache.keys() or date in synthetic_row_cache.keys():
            continue

        # Adding a synthetic marker
        row = get_latest_metrics(date, min_date, row_cache, synthetic_row_cache)

        # If no row is found, then code coverage is either too recent (today) or too old. 
        if row == None:
            continue

        row["RowKey"] = str(partition_key) + time_to_str(date)
        metric = create_metric(row, date)
        print(metric)
        print("Inserting " + str(metric))
        try:
            table_service.insert_entity(table_name, metric)
            print(f"Successfully inserted metrics in table '{table_name}'.")
        except:
            print(f"Failed to insert metrics in '{table_name}'.")
            print(sys.exc_info()[0], " occurred.")


def process_all_rows(table_service):
    # Get list of all partition keys. PKey have the 'Project-DefinitionId' format.
    pkey_list = get_partition_key_list(table_service)

    for pkey in pkey_list:
        process_rows_from_partition(table_service, pkey)

def run_clean_process():
    table_service = get_table_service()
    process_all_rows(table_service)

if __name__ == "__main__":
    run_clean_process()