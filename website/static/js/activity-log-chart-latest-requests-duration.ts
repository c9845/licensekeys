/**
 * activity-log-chart-latest-requests-duration.ts
 * 
 * This builds the chart that shows the time the app took to respond to the latest 
 * requests. This chart is useful for identifying recent latency increases or if
 * specific request/endpoints are very slow.
 */

/// <reference path="common.ts" />

//chart is stored outside of Vue object to prevent "maximum call stack size exceeded".
let activityDurationChart = undefined;

if (document.getElementById("activityLogChartLatestRequestsDuration")) {
    //@ts-ignore cannot find name Vue
    var activityLogChartLatestRequestsDuration = new Vue({
        name: 'activityLogChartLatestRequestsDuration',
        delimiters: ['[[', ']]'],
        el: '#activityLogChartLatestRequestsDuration',
        data: {
            showHelp: false,

            //Raw data to build chart with.
            rawData: [] as activityLog[],

            //Errors.
            msg: "",
            msgType: "",

            //endpoints
            urls: {
                latestDur: "/api/activity-log/latest-requests-duration/",
            },
        },
        methods: {
            //getData retrieves the data from the server.
            getData: function () {
                this.msg = "Retrieving data, this may take a while...";
                this.msgType = msgTypes.primary;

                let reqParams: Object = {
                    ignorePDF: true,
                };
                fetch(get(this.urls.latestDur, reqParams))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then(function (j) {
                        //check if response is an error from the server
                        let err: string = handleAPIErrors(j);
                        if (err !== '') {
                            activityLogChartLatestRequestsDuration.msg = err;
                            activityLogChartLatestRequestsDuration.msgType = msgTypes.danger;
                            return;
                        }

                        //Save data for charting.
                        activityLogChartLatestRequestsDuration.rawData = j.Data || [];
                        activityLogChartLatestRequestsDuration.msg = "";

                        //Build the chart.
                        activityLogChartLatestRequestsDuration.buildChart();

                        return;
                    })
                    .catch(function (err) {
                        console.log("fetch() error: >>", err, "<<");
                        activityLogChartLatestRequestsDuration.msg = 'An unknown error occured. Please try again.';
                        activityLogChartLatestRequestsDuration.msgType = msgTypes.danger;
                        return;
                    });

                return;
            },

            //buildChart takes the raw data retrieved in getData() and builds and 
            //shows the chart.
            buildChart: function () {
                //Where chart will be shown.
                const elemID: string = 'activity-request-duration-chart';
                var ctx: HTMLElement = document.getElementById(elemID);

                //Get the data points to chart.
                //get the data points to chart
                let xAxisLabels = [];
                let yAxisPoints = [];
                for (let r of (this.rawData as activityLog[])) {
                    xAxisLabels.push(r.DatetimeCreated);
                    yAxisPoints.push(r.TimeDuration);
                }

                //Set chart options.
                let ops = {
                    legend: {
                        display: false, //don't show legend since we will describe data ourselves; legend is a bit messy
                    },
                    scales: {
                        y: {
                            type: "linear",
                            position: "left",
                            ticks: {
                                min: 0,
                                maxTicksLimit: 10, //doesn't work?
                            }
                        },
                    },
                    animation: {
                        duration: 0, //don't animate chart when it is shown for better performance
                    },
                    plugins: {
                        tooltip: {
                            callbacks: {
                                //title modifies the tooltip to show the URL as the title instead of the date
                                //Don't really know why we need the [0] index here b/c we don't need it for labels.
                                title: function (context) {
                                    let index: number = context[0].dataIndex;
                                    let point = activityLogChartLatestRequestsDuration.rawData[index];
                                    return point.Method + ": " + point.URL;
                                },
    
                                //this modifies the tooltip label to show the produced lot number or container code when a point is hovered
                                //Showing this extra info is helpful for figuring out what batch/raw material a point belongs to for diagnosing issues.
                                label: function (context) {
                                    let index: number = context.dataIndex;
                                    let point = activityLogChartLatestRequestsDuration.rawData[index];
                                    return point.DatetimeCreated + " (" + point.TimeDuration + "ms)";
                                }
                            }
                        }
                    }
                };

                //Generate the chart.
                //@ts-ignore cannot find name Chart
                activityDurationChart = new Chart(ctx, {
                    type: 'bar',
                    data: {
                        labels: xAxisLabels,
                        datasets: [
                            {
                                label: "Duration (ms)",
                                backgroundColor: "#007bff", //same as bootstrap primary
                                data: yAxisPoints,
                                fill: false,  //no grey under lines, just show lines, more clear for users
                                lineTension: 0,      //same as everwhere else in app
                            },
                        ]
                    },
                    options: ops,
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
