/**
 * activity-log-chart-duration-by-endpoint.ts
 * 
 * This builds the table that shows the max/min/avg duration it took to respond to
 * each endpoint.
 */

/// <reference path="common.ts" />

interface durationByEndpoint {
    URL: string,
    EndpointHits: number,
    Method: string,
    AverageTimeDuration: number,
    MaxTimeDuration: number,
    MinTimeDuration: number,
}

if (document.getElementById("activityLogChartDurationByEndpoint")) {
    //@ts-ignore cannot find name Vue
    var activityLogChartDurationByEndpoint = new Vue({
        name: 'activityLogChartDurationByEndpoint',
        delimiters: ['[[', ']]'],
        el: '#activityLogChartDurationByEndpoint',
        data: {
            showHelp: false,

            //Raw data to build chart with.
            rawData: [] as durationByEndpoint[],

            //Errors.
            msg: "",
            msgType: "",

            //endpoints
            urls: {
                maxAvgDuration: "/api/activity-log/duration-by-endpoint/",
            },
        },
        methods: {
            //getData retrieves the data from the server.
            getData: function () {
                this.msg = "Retrieving data, this may take a while...";
                this.msgType = msgTypes.primary;

                //Check if we should ignore PDF creation from data retrieved since
                //PDF creation takes a while causing data to be skewed.
                let sp: URLSearchParams = new URLSearchParams(document.location.search);
                let ignore: boolean = true;
                if (sp.has("ignorePDF") && sp.get("ignorePDF") === "false") {
                    ignore = false
                }

                //Make API request.
                let reqParams: Object = {
                    ignorePDF: ignore,
                };
                fetch(get(this.urls.maxAvgDuration, reqParams))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then(function (j) {
                        //check if response is an error from the server
                        let err: string = handleAPIErrors(j);
                        if (err !== '') {
                            activityLogChartDurationByEndpoint.msg = err;
                            activityLogChartDurationByEndpoint.msgType = msgTypes.danger;
                            return;
                        }

                        //Save data for charting.
                        activityLogChartDurationByEndpoint.rawData = j.Data || [];
                        activityLogChartDurationByEndpoint.msg = "";

                        //Build the chart.
                        activityLogChartDurationByEndpoint.buildChart();

                        return;
                    })
                    .catch(function (err) {
                        console.log("fetch() error: >>", err, "<<");
                        activityLogChartDurationByEndpoint.msg = 'An unknown error occurred. Please try again.';
                        activityLogChartDurationByEndpoint.msgType = msgTypes.danger;
                        return;
                    });

                return;
            },
        },
        mounted() {
            //Make request to get data, which will then build the chart.
            this.getData();
            return;
        }
    });
}
