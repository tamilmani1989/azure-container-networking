import sys, csv
from datetime import datetime
from xml.etree import ElementTree
from xml.dom import minidom
from xml.etree.ElementTree import Element, SubElement, Comment
import statistics
import math

# Indices to access columns of results.jtl
TIMESTAMP = 0
ELAPSED = 1
LABEL = 2
RESPONSE_CODE = 3
RESPONSE_MESSAGE = 4
THREAD_NAME = 5
SUCCESS = 7
FAILURE_MESSAGE = 8
BYTES = 9
URL = 13
LATENCY = 14

def prettify(elem):
	"""Returns a pretty-printed XML string for the Element.
	"""
	rough_string = ElementTree.tostring(elem, 'utf-8')
	reparsed = minidom.parseString(rough_string)
	return reparsed.toprettyxml(indent="  ")

def retrieve_jmeter_results(jmeter_file):
	"""Returns a list of JMeter JTL rows without header.
	The JMeter JTL file must be in CSV format.
	Average time to create a container (from create call to success status)
	"""
	csv_reader = csv.reader(jmeter_file)
	next(csv_reader)
	return list(csv_reader)

def calculate_percentile(nums, p):
	"""Returns p-th percentile of number list nums
	"""
	N = p/100.0 * (len(nums) + 1)
	high_index = math.ceil(N)
	low_index = math.floor(N)
	if high_index > len(nums)-1:
		return nums[len(nums)-1]
	return (nums[high_index] + nums[low_index])/2
	pass

def fetch_statistics(jmeter_results, request_label, success_label):
	"""Returns and logs statistics for operation specified by 'request' and 'success' labels
	Times measured in seconds
	"""
	print("\n############################# {} RESULTS #############################".format(request_label))
	timestamps = {}
	workers = {}
	succeeded_operation_times = []

	# Iterate through JMeter output
	for result in jmeter_results:
		label = result[THREAD_NAME]
		worker_id, _, call_number = label.split("-")
		response = result[RESPONSE_CODE]

		# To Track Worker Request Success
		if worker_id not in workers:
			workers[worker_id] = {}
			workers[worker_id]["denied_requests"] = 0
			workers[worker_id]["accepted_requests"] = 0

		# Operation Request Sent
		if request_label in result[LABEL]:
			if response[0:8] == "Non HTTP":
				workers[worker_id]["denied_requests"] += 1
				if "first_denied_request" not in workers[worker_id]:
					workers[worker_id]["first_denied_request"] = "Worker {} received first timeout response after {} calls to {}. \n Full error message was: {}".format(worker_id, call_number, request_label, response)
			else:
				workers[worker_id]["accepted_requests"] += 1
			timestamps[label] = {}
			timestamps[label]["start_timestamp"] = result[TIMESTAMP]
			first_timestamp = result[TIMESTAMP] if len(timestamps) == 1 else first_timestamp
		
		# Operation Completed Successfully
		if success_label in result[LABEL]:
			operation_time = (int(result[TIMESTAMP]) - int(timestamps[label]["start_timestamp"]))/1000.0
			succeeded_operation_times.append(operation_time)
			last_timestamp = result[TIMESTAMP]
	
	# Return and log results
	try:
		time_window = (int(last_timestamp)-int(first_timestamp))/1000.0
		num_operations_total = len(timestamps)
		num_operations_succeeded = len(succeeded_operation_times)
		stddev = statistics.pstdev(succeeded_operation_times)
		percentile_50th = calculate_percentile(succeeded_operation_times, 50)
		percentile_90th = calculate_percentile(succeeded_operation_times, 90)  # 90% of the Data Falls Below
		throughput = num_operations_succeeded/time_window # ex: Number of Containers Created Per Second
		average_completion_time = sum(succeeded_operation_times)/num_operations_succeeded
		success_ratio = float(num_operations_succeeded)/float(num_operations_total)

		print("\n--------- Results ---------\n")
		print("In total, {} requests to {} were sent and {} were accepted and reached completed status. ".format(num_operations_total, request_label, num_operations_succeeded))
		print("{} {}s accepted and successfully completed in total {} seconds, giving a success rate of {} percent!".format(num_operations_succeeded, request_label, time_window, success_ratio*100))
		print("The average successful {} moved from request to completed status in {} seconds".format(request_label, average_completion_time))
		print("On average {} {}s were completed per second".format(request_label, throughput))
		print("50th Percentile for {} completion time: {}".format(request_label, percentile_50th))
		print("90th Percentile for {} completion time: {}".format(request_label, percentile_90th))
		print("Standard Deviation for {} completion time: {}".format(request_label, stddev))

		print("\n--------- Worker Reports ---------")
		for worker_id in workers:
			print("\nWorker {} made {} {} calls. {} of these calls were denied with a timeout error".format(worker_id, workers[worker_id]["accepted_requests"] + workers[worker_id]["denied_requests"], request_label, workers[worker_id]["denied_requests"]))
			if "first_denied_request" in workers[worker_id]:
				print(workers[worker_id]["first_denied_request"])
		return {
			'average_completion_time':average_completion_time,
			'success_ratio':success_ratio,
			'percentile_50th': percentile_50th,
			'percentile_90th': percentile_90th,
			'stddev': stddev
		}
	# Catch exceptions (such as no requests sent/succeeded)
	except:
		print("\nNo {} requests were successfully completed.".format(request_label))
		print("\n--------- Worker Reports ---------")
		for worker_id in workers:
			print("\nWorker {} made {} {} calls. {} of these calls were denied with a timeout error".format(worker_id, workers[worker_id]["accepted_requests"] + workers[worker_id]["denied_requests"], request_label, workers[worker_id]["denied_requests"]))
			if "first_denied_request" in workers[worker_id]:
				print(workers[worker_id]["first_denied_request"])
		return {
			'average_completion_time':1000,
			'success_ratio':0,
			'percentile_50th': 1000,
			'percentile_90th': 1000,
			'stddev': 0
		}

def create_request_attrib(jmeter_result):
	"""Returns a JSON with attributes for a JUnit testsuite: https://llg.cubic.org/docs/junit/
	The JMeter JTL file must be in CSV format.
	"""
	request_time = float(jmeter_result[ELAPSED])/1000.0

	return {
		'name': jmeter_result[LABEL],
		'time': str(request_time),
		'error_message': jmeter_result[RESPONSE_MESSAGE]
	}

def create_test_case_attrib(request_attrib):
	"""Returns a JSON with attributes for a JUnit testcase: https://llg.cubic.org/docs/junit/
	The JMeter JTL file must be in CSV format.
	"""
	return {
		'classname': 'httpSample',
		'name': request_attrib['name'],
		'time': request_attrib['time']
}

def create_assertion_test_suite_attrib(junit_results):
	"""Returns a JSON with attributes for JUnit testsuite: https://llg.cubic.org/docs/junit/
	The JMeter JTL file must be in CSV format.
	"""
	return {
		'id': '1',
		'name': 'Load Test Assertions',
		'package': 'load test',
		'hostname': 'Azure DevOps',
		'time': str(junit_results['time']),
		'tests': str(junit_results['tests']),
		'failures': str(len(junit_results['requests']['failures'])),
		'errors': '0'
	}

def create_test_suite_attrib(junit_results):
	"""Returns a JSON with attributes for JUnit testsuite: https://llg.cubic.org/docs/junit/
	The JMeter JTL file must be in CSV format.
	"""
	return {
		'id': '2',
		'name': 'HTTP Requests',
		'package': 'load test',
		'hostname': 'Azure DevOps',
		'time': str(junit_results['time']),
		'tests': str(junit_results['tests']),
		'failures': str(len(junit_results['requests']['failures'])),
		'errors': '0'
	}

def create_error_test_case_attrib(error_message):
	"""Returns a JSON with attributes for JUnit testcase for failed requests: https://llg.cubic.org/docs/junit/
	The JMeter JTL file must be in CSV format.
	"""
	return {
		'message': error_message,
		'type': 'exception'
	}

def requests(jmeter_results):
	"""Returns a JSON with successful and failed HTTP requests.
	The JMeter JTL file must be in CSV format.
	"""
	failed_requests = []
	successful_requests = []

	for result in jmeter_results:
		request_attrib = create_request_attrib(result)

		if SUCCESS in result and result[SUCCESS] == 'true':
			successful_requests.append(request_attrib)
		else:
			failed_requests.append(request_attrib)

	return {
		'success': successful_requests,
		'failures': failed_requests
	}

def total_time_seconds(jmeter_results):
	"""Returns the total test duration in seconds.
	The JMeter JTL file must be in CSV format.
	"""
	max_timestamp = max(jmeter_results, key=lambda result: int(result[TIMESTAMP]))
	min_timestamp = min(jmeter_results, key=lambda result: int(result[TIMESTAMP]))
	total_timestamp = int(max_timestamp[TIMESTAMP]) - int(min_timestamp[TIMESTAMP])

	return float(total_timestamp)/1000.0

def create_junit_results(jtl_results_filename):

	with open(jtl_results_filename) as jmeter_file:
		jmeter_results = retrieve_jmeter_results(jmeter_file)
		time = total_time_seconds(jmeter_results)

		# Initialize operation statistics and set testing thresholds
		create_container_statistics = fetch_statistics(jmeter_results, "Create Container", "Container Created")
		create_container_statistics["threshold_50"] = float(sys.argv[9])
		create_container_statistics["threshold_90"] = float(sys.argv[10])
		create_container_statistics["threshold_avg"] = float(sys.argv[11])

		delete_container_statistics = fetch_statistics(jmeter_results, "Delete Container", "Container Deleted")
		delete_container_statistics["threshold_50"] = float(sys.argv[12])
		delete_container_statistics["threshold_90"] = float(sys.argv[13])
		delete_container_statistics["threshold_avg"] = float(sys.argv[14])
		
		create_network_statistics = fetch_statistics(jmeter_results, "Create Network", "Network Created")
		create_network_statistics["threshold_50"] = float(sys.argv[3])
		create_network_statistics["threshold_90"] = float(sys.argv[4])
		create_network_statistics["threshold_avg"] = float(sys.argv[5])

		create_subnet_statistics = fetch_statistics(jmeter_results, "Create Subnet", "Subnet Created")
		create_subnet_statistics["threshold_50"] = float(sys.argv[6])
		create_subnet_statistics["threshold_90"] = float(sys.argv[7])
		create_subnet_statistics["threshold_avg"] = float(sys.argv[8])


		return {
			'tests': len(jmeter_results),
			'time': time,
			'requests': requests(jmeter_results),
			'statistics': {
				'Create Container' : create_container_statistics, 
				'Delete Container': delete_container_statistics,
				'Create Network' : create_network_statistics, 
				'Create Subnet': create_subnet_statistics,
				}
		}

def create_properties(test_suite):
	"""Creates a JUnit properties element for testsuite: https://llg.cubic.org/docs/junit/
	The JMeter JTL file must be in CSV format.
	"""
	return SubElement(test_suite, 'properties')


def create_test_suite(test_suites, junit_results):
	"""Creates JUnit testsuite: https://llg.cubic.org/docs/junit/
	Assertion Tests to flag anf fail abnormal behavior of DNC
	"""

	test_suite_attrib = create_assertion_test_suite_attrib(junit_results)
	test_suite = SubElement(test_suites, 'testsuite', test_suite_attrib)

	create_properties(test_suite)

	for operation in junit_results['statistics']:
		average_completion_time = junit_results['statistics'][operation]["average_completion_time"]
		percentile_50th = junit_results['statistics'][operation]["percentile_50th"]
		percentile_90th = junit_results['statistics'][operation]["percentile_90th"]
		stddev = junit_results['statistics'][operation]["stddev"]
		success_ratio = junit_results['statistics'][operation]["success_ratio"]

		if average_completion_time > junit_results['statistics'][operation]['threshold_avg']:
			create_failed_test_case(test_suite, {'name': "Average Time for {} Between Request and Completion Status".format(operation), 'time': str(average_completion_time), 'error_message': 'Average time for {} is longer than expected {} seconds.'.format(operation, junit_results['statistics'][operation]['threshold_avg'])})
		else:
			create_successful_test_case(test_suite, {'name': "Average Time for {} Between Request and Completion Status".format(operation), 'time': str(average_completion_time), 'error_message': ''})
		
		if percentile_50th > junit_results['statistics'][operation]['threshold_50']:
			create_failed_test_case(test_suite, {'name': '{} Time: 50% Percentile'.format(operation), 'time': str(percentile_50th), 'error_message': '50th percentile for {} is longer than expected {} seconds.'.format(operation, junit_results['statistics'][operation]['threshold_50'])})
		else:
			create_successful_test_case(test_suite, {'name': '{} Time: 50% Percentile'.format(operation), 'time': str(percentile_50th), 'error_message': ''})

		if percentile_90th > junit_results['statistics'][operation]['threshold_90']:
			create_failed_test_case(test_suite, {'name': '{} Time: 90% Percentile'.format(operation), 'time': str(percentile_90th), 'error_message': '90th percentile for {} is longer than expected {} seconds.'.format(operation, junit_results['statistics'][operation]['threshold_90'])})
		else:
			create_successful_test_case(test_suite, {'name': '{} Time: 90% Percentile'.format(operation), 'time': str(percentile_90th), 'error_message': ''})
		
		if success_ratio < 1:
			create_failed_test_case(test_suite, {'name': 'Lower than expected success rate for {}'.format(operation), 'time': '0', 'error_message': 'Only {} percent of {} were successfully completed'.format(success_ratio*100, operation)})
		else:
			create_successful_test_case(test_suite, {'name': 'All {} Operations Successful!'.format(operation), 'time': '1', 'error_message': ''})


def create_failed_test_case(test_suite, failed_request):
	"""Creates a JUnit test case for failed HTTP requests: https://llg.cubic.org/docs/junit/
	The JMeter JTL file must be in CSV format.
	"""
	test_case_attrib = create_test_case_attrib(failed_request)
	error_test_case_attrib = create_error_test_case_attrib(failed_request['error_message'])

	test_case = SubElement(test_suite, 'testcase', test_case_attrib)
	test_case_error = SubElement(test_case, 'error', error_test_case_attrib)

def create_successful_test_case(test_suite, successful_request):
	"""Creates a JUnit test case for successful HTTP requests: https://llg.cubic.org/docs/junit/
	The JMeter JTL file must be in CSV format.
	"""
	test_case_attrib = create_test_case_attrib(successful_request)
	test_case = SubElement(test_suite, 'testcase', test_case_attrib)

def create_test_suites(jtl_results_filename):
	"""Creates a JUnit testsuites element: https://llg.cubic.org/docs/junit/
	The JMeter JTL file must be in CSV format.
	"""
	test_suites = Element('testsuites')
	junit_results = create_junit_results(jtl_results_filename)
	create_test_suite(test_suites, junit_results)

	return prettify(test_suites)

def main():
  print('Processing JMeter Output and Generating JUnit Test Results...')
  thresholds = {
	  "container_"

  }

  with open(sys.argv[2], "w") as output_file:
    test_suites = create_test_suites(sys.argv[1])
    output_file.write(test_suites)

  print("\n############################# JUnit Test Results Generated #############################")

if __name__ == '__main__':
	main()
