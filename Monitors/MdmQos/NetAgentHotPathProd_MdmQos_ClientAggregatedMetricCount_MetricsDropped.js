var threshold75Percent = 0.75;
var threshold100Percent = 1.0;
var result = {};
var dimensions = {};

try{
    var limitSeries = fetchMetrics(
        "NetAgentHotPathProd",
        "MdmQos",
        "ClientAggregatedMetricCountLimit",
        "Sum",
        dimensions
    );
}
catch(err){
    // do nothing
}

result.evaluatedValue = 0;
result.thresholdViolated = false;
result.severity = 4;
result.message = "ClientAggregatedMetricCountLimit does not exist.";

if(limitSeries && sum(limitSeries)){

    var meanVal = 0;
    if (timeSeries.Sum.length>0)
    {
      var total=0;
      for(var i in timeSeries.Sum) 
      { 
        total += timeSeries.Sum[i]; 
      }
      
      meanVal = (total / timeSeries.Sum.length) || 0;
    }
    
    var maxLimit = max(limitSeries);
    result.evaluatedValue = meanVal / (maxLimit || 1);
    result.thresholdViolated = result.evaluatedValue > threshold75Percent;
	
	var msg1 = "This in an automatic alert from MDM. Warning: You are currently using more than ";
	var msg2 = " of your allocated quota of events per minute. If you go above 100%, your account will " +
			"be sampled and some data will be lost. ";
	var msg3 = "Your current usage is " + meanVal + " out of " + maxLimit + ". " +
			"This might mean you are sending too many metrics from client machines: too many " +
			"metrics per machine and/or too many machines are sending metrics." +
			"If this is unexpected, please make sure you are not emitting more than needed. " +
			"Otherwise, you can request an increase to your current limit by visiting: " +
			"https://jarvis.dc.ad.msft.net/?page=settings&mode=mdm&tab=account&section=accountLimitRequest&account=NetAgentHotPathProd ." +
			"This monitor can be disabled or customized " +
			"at Manage -> Geneva Hot Path -> NetAgentHotPathProd -> MdmQos -> ClientAggregatedMetricCount -> " +
			"Monitors. You can see your usage and more information on automatic monitoring and limits by " +
			"clicking on the links at the bottom of the description.";


	if(result.evaluatedValue > threshold100Percent){
			result.severity = 2;
			result.message = "This in an automatic alert. Warning: You are currently using more than your " +
				"allocated quota of events per minute. Your account is being sampled: data loss is occurring. " + msg3;
	}
    else if(result.evaluatedValue > threshold75Percent){
        result.severity = 3;
        result.message = msg1 + "75%" + msg2 + msg3;
    }
	else{
		result.message = "Your current usage is " + meanVal + " out of " + maxLimit + ".";
	}
}

return stringify(result);