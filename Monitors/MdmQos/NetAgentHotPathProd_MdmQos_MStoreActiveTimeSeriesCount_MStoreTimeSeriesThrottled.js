var result = {};
result.thresholdViolated = false;

if (timeSeries.Sum.length > 0)
{
    var threshold85Percent = 0.85;
    var threshold95Percent = 0.95;

    timeSeries.Sum.sort(function(a, b){ return a - b });
    var activeTimeSeriesCountMedian = timeSeries.Sum[Math.floor(timeSeries.Sum.length / 2)] || 0;

    if (activeTimeSeriesCountMedian > 0)
    {
        var droppedSamplesCount = 0;
        try
        {
            var droppedSamplesCountValues = fetchMetrics(
                "NetAgentHotPathProd",
                "MdmQos",
                "MStoreDroppedSamplesCount",
                "Sum",
                {"Reason" : "AccountThrottled"}
            );

            // We do not take into account first 4 minutes of dropped samples because
            // when limit is increased it takes several minutes to propagate across
            // all FrontEnd and mStore machines, so they may continue throttling
            droppedSamplesCount = sum(droppedSamplesCountValues.slice(4)) || 0;
        }
        catch(err)
        {
            // do nothing
        }

        var timeSeriesLimit = 0;
        var timeSeriesLimitIncreasing = false;
        try
        {
            var timeSeriesLimitValues = fetchMetrics(
                "NetAgentHotPathProd",
                "MdmQos",
                "MStoreActiveTimeSeriesLimit",
                "Sum",
                {}
            );

            var startTimeSeriesLimit = ((timeSeriesLimitValues[0] + timeSeriesLimitValues[1]) / 2) || 0;
            var endTimeSeriesLimit = ((timeSeriesLimitValues[timeSeriesLimitValues.length - 1] + timeSeriesLimitValues[timeSeriesLimitValues.length - 2]) / 2) || 0;
            timeSeriesLimitIncreasing = startTimeSeriesLimit < endTimeSeriesLimit;
            timeSeriesLimit = max(timeSeriesLimitValues) || 0;
        }
        catch(err)
        {
            // do nothing
        }

        var instructionsMessage = "Please visit MdmQos dashboard https://jarvis-west.dc.ad.msft.net/dashboard/NetAgentHotPathProd/GenevaQos/%25E2%2586%2590%2520MdmQos " +
                                  "to check current number of time series, limits and number of dropped samples. " +
                                  "If the increase in number of active time series is expected, request an increase to your current limit by visiting: " +
                                  "https://jarvis.dc.ad.msft.net/?page=settings&mode=mdm&tab=account&section=accountLimitRequest&account=NetAgentHotPathProd."
                                  "If the increase is not expected, use MdmQos dashboard to find namespaces and metrics which started to emit unexpected " +
                                  "load and disable corresponding preaggregates. " + 
                                  "To learn more about MDM time series throttling visit " + 
                                  "https://jarvis-west.dc.ad.msft.net/?page=documents&section=b7a73824-bbbf-49fc-8c3e-a97c27a7659e&id=d1c2618e-b4f3-4990-b90f-783b594f3c4a. " +
                                  "This monitor can be disabled or customized at: " +
                                  "https://jarvis-west.dc.ad.msft.net/?page=settings&mode=mdm&tab=monitors&account=NetAgentHotPathProd&namespace=MdmQos&metric=MStoreActiveTimeSeriesCount&monitor=MStoreTimeSeriesThrottled.";

        if (droppedSamplesCount > 0)
        {
            result.thresholdViolated = true;
            result.evaluatedValue = droppedSamplesCount;

            var droppedSamplesMessage = "Number of dropped samples for last 10 minutes is " + droppedSamplesCount + ". ";
            if (timeSeriesLimit === 0)
            {
                result.severity = 3;
                result.message = "There are multiple drops of monitoring account " + env.monitoringAccount + " samples because new time series are throttled by MDM. " +
                                 "This usually happens when the account's active time series limit is lower than actual number of active time series. " +
                                 droppedSamplesMessage + instructionsMessage;
            }
            else if (timeSeriesLimit < activeTimeSeriesCountMedian)
            {
                result.severity = 3;
                result.message = "The number of active time series in account " + env.monitoringAccount + " has exceeded the limit. Current: " + activeTimeSeriesCountMedian +
                                 " Limit: " + timeSeriesLimit + ". New time series are throttled and data for them is dropped. " + droppedSamplesMessage + instructionsMessage;
            }
            else if (timeSeriesLimitIncreasing)
            {
                // This covers the case when customer was throttled correctly, requested time series limit increase
                // and we have approved this request. Then for short period (while new limits are still propagated
                // and monitor still sees dropped samples due to lookback) we may see dropped samples while limit is
                // greater than number of active time series, but we don't want to alert on it
            }
            else
            {
                result.severity = 3;
                result.message = "There are multiple drops of monitoring account " + env.monitoringAccount + " samples because new time series are throttled by MDM. " +
                                 droppedSamplesMessage + "Number of active time series does not exceed the limit set in the monitoring account. Please escalate to " +
                                 "Geneva MDM team (http://jarvis-west.dc.ad.msft.net?support=true).";
            }
        }
        else if (timeSeriesLimit > 0)
        {
            var exceedingLimitMessage = "Current: " + activeTimeSeriesCountMedian + " Limit: " + timeSeriesLimit + ". Exceeding the limit may result in new time series being " +
                                        "throttled and data for them dropped, which can cause a loss of monitoring capabilities. No data loss is occurring at this time. " + instructionsMessage;

            if (timeSeriesLimit * threshold95Percent < activeTimeSeriesCountMedian)
            {
                result.thresholdViolated = true;
                result.evaluatedValue = activeTimeSeriesCountMedian / timeSeriesLimit;
                result.severity = 3;
                result.message = "The number of active time series in account " + env.monitoringAccount + " is 95% of the limit. " + exceedingLimitMessage;
            }
            else if (timeSeriesLimit * threshold85Percent < activeTimeSeriesCountMedian)
            {
                result.thresholdViolated = true;
                result.evaluatedValue = activeTimeSeriesCountMedian / timeSeriesLimit;
                result.severity = 3;
                result.message = "The number of active time series in account " + env.monitoringAccount + " is 85% of the limit. " + exceedingLimitMessage;
            }
        } 
    }
}

return stringify(result);