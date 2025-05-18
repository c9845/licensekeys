/**
 * activity-log-chart-latest-requests-duration.ts
 * 
 * This builds the chart that shows the time the app took to respond to the latest 
 * requests. This chart is useful for identifying recent latency increases or if
 * specific request/endpoints are very slow.
 */

import { createApp } from "vue";
import { Chart, ChartOptions } from "chart.js/auto"; //auto just makes importing easier.
import { msgTypes, apiBaseURL } from "./common";
import { get, handleRequestErrors, getJSON, handleAPIErrors } from "./fetch";

//chart is stored outside of Vue object to prevent "maximum call stack size exceeded".
let activityDurationChart = undefined;

var durationLatestRequests: any; //must be "any", not "ComponentPublicInstance" to remove errors when calling functions (methods) of this Vue instance.
if (document.getElementById("activityLogChartLatestRequestsDuration")) {
    durationLatestRequests = createApp({
        name: 'activityLogChartLatestRequestsDuration',

        compilerOptions: {
            delimiters: ["[[", "]]"],
        },

        data() {
            return {
                //Show help, description of what chart is showing.
                showHelp: false,

                //Raw data to build chart with.
                rawData: [] as activityLog[],

                //Errors.
                msg: "",
                msgType: "",

                //endpoints
                urls: {
                    latestDur: apiBaseURL + "activity-log/latest-requests-duration/",
                },
            }
        },
        methods: {
            //getData retrieves the data from the server.
            getData: function () {
                this.msg = "Retrieving data, this may take a while...";
                this.msgType = msgTypes.primary;

                //Make API request.
                let reqParams: Object = {};
                fetch(get(this.urls.latestDur, reqParams))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then((j) => {
                        //Check if response is an error from the server.
                        let err: string = handleAPIErrors(j);
                        if (err !== "") {
                            this.msg = err;
                            this.msgType = msgTypes.danger;
                            return;
                        }

                        //Save data for charting.
                        this.rawData = j.Data || [];
                        this.msg = "";

                        //Build the chart.
                        this.buildChart();

                        return;
                    })
                    .catch((err) => {
                        console.log("fetch() error: >>", err, "<<");
                        this.msg = "An unknown error occurred. Please try again.";
                        this.msgType = msgTypes.danger;
                        return;
                    });

                return;
            },

            //buildChart takes the raw data retrieved in getData() and builds and 
            //shows the chart.
            buildChart: function () {
                //Where chart will be shown.
                const elemID: string = 'activity-request-duration-chart';
                var ctx: HTMLElement = document.getElementById(elemID)!;

                //Get the data points to chart.
                let xAxisLabels = [];
                let yAxisPoints = [];
                for (let r of (this.rawData as activityLog[])) {
                    xAxisLabels.push(r.DatetimeCreated);
                    yAxisPoints.push(r.TimeDuration);
                }

                //Set chart options.
                let ops: ChartOptions = {
                    scales: {
                        y: {
                            type: "linear",
                            min: 0,
                            position: "left",
                            ticks: {
                                maxTicksLimit: 10,
                            }
                        },
                        x: {
                            //Don't show the x-axis, it will end up just being dates anway
                            //and most likely, all the same date since the activity shown
                            //is the most recent and typically there is a lot.
                            display: false,
                        }
                    },
                    animation: {
                        duration: 0, //don't animate for improved performance
                    },
                    plugins: {
                        legend: {
                            display: false,
                        },
                        tooltip: {
                            callbacks: {
                                //title modifies the tooltip to show the URL as the title 
                                //instead of the date (x axis label). Showing the URL is
                                //more useful to us so that we can identify why endpoint
                                //took so long to respond.
                                //
                                title: function (context: any) {
                                    //Get index of data hovered on.
                                    //
                                    //Don't really know why we need the [0] index here
                                    //since there always seems to be only one element in
                                    //the array.
                                    let index: number = context[0].dataIndex;

                                    //Look up matching data from data we retreived from
                                    //the database.
                                    let data: activityLog = durationLatestRequests.rawData[index];

                                    //Build the test we want to display
                                    return data.Method + ": " + data.URL;
                                },
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
    }).mount("#activityLogChartLatestRequestsDuration");
}
