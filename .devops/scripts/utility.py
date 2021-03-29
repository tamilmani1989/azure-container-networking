import datetime
import json

def str_to_time(timeStr):
    timeStr = timeStr.split(".")[0]
    if("Z" not in timeStr):
        timeStr += "Z"
    return datetime.datetime.strptime(timeStr, "%Y-%m-%dT%H:%M:%SZ")

def time_to_str(time):
    return datetime.datetime.strftime(time, "%Y-%m-%dT%H:%M:%SZ")

def pretty_print(json_object):
    print(json.dumps(json_object, indent=2))