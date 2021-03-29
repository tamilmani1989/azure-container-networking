# Driver script used for the code coverage publishing pipeline.
import argparse
import datetime
from publish_build_metrics import publish_range_build_metrics, time_to_str

# Example usage python publish_build_metrics_driver.py -args msdata:vienna:9169-8475 responsibleai:interpret-text:66 -publish console
def driver(pipeline_info, min_finish_time_utc, max_finish_time_utc, target):
    pipeline_info = pipeline_info.split(':')
    organisation = pipeline_info[0]
    project = pipeline_info[1]
    definition_ids = pipeline_info[2].split('-')

    publish_range_build_metrics(min_finish_time_utc, max_finish_time_utc, definition_ids, target, organisation, project)

if __name__ == "__main__":
  parser = argparse.ArgumentParser(description='Publishes build metrics of a given DevOps build run.')
  parser.add_argument('-publish'      , type=str, help="Where to publish the metrics, either \"storage\" or \"console\".", default="console")
  parser.add_argument('-minFinishTime', type=str, help="Filters builds that finished after this date.")
  parser.add_argument('-maxFinishTime', type=str, help="Filters builds that finished before this date.")
  parser.add_argument('-pipelines' , type=str, nargs="*", help="A list of triples with format organisation:project:builddef1-builddef2... .", required=True)
  ns = parser.parse_args()
  target = ns.publish
  max_finish_time_utc = ns.maxFinishTime
  min_finish_time_utc = ns.minFinishTime
  if not max_finish_time_utc:
      max_finish_time_utc = time_to_str(datetime.datetime.utcnow()- datetime.timedelta(days=5))
  if not min_finish_time_utc:
      min_finish_time_utc = time_to_str(datetime.datetime.utcnow() - datetime.timedelta(days=20))
  pipelines = ns.pipelines
  for pipeline_info in pipelines:
    driver(pipeline_info, min_finish_time_utc, max_finish_time_utc, target)