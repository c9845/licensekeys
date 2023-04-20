/**
 * activity-log-chart-max-and-avg-monthly-duration.ts
 * 
 * This builds the chart that shows the max and average time the app took to respond
 * to requests per month. This chart is useful for identifying if the app is getting
 * slower over time.
 */

/// <reference path="common.ts" />

//chart is stored outside of Vue object to prevent "maximum call stack size exceeded".
let reportActivityMaxAvgDurationChart = undefined;

interface activityMaxAvgDuration {
    AverageTimeDuration: number,
    MaxTimeDuration: number,
    Year: number,
    Month: number,
}

if (document.getElementById("activityLogChartMaxAndAvgMonthlyDuration")) {
    //@ts-ignore cannot find name Vue
    var activityLogChartMaxAndAvgMonthlyDuration = new Vue({
        name: 'activityLogChartMaxAndAvgMonthlyDuration',
        delimiters: ['[[', ']]'],
        el: '#activityLogChartMaxAndAvgMonthlyDuration',
        data: {
            showHelp: false,

            //Raw data to build chart with.
            rawData: [] as activityMaxAvgDuration[],

            //Errors.
            msg: "",
            msgType: "",

            //endpoints
            urls: {
                maxAvgDuration: "/api/activity-log/max-and-avg-monthly-duration/",
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
                fetch(get(this.urls.maxAvgDuration, reqParams))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then(function (j) {
                        //check if response is an error from the server
                        let err: string = handleAPIErrors(j);
                        if (err !== '') {
                            activityLogChartMaxAndAvgMonthlyDuration.msg = err;
                            activityLogChartMaxAndAvgMonthlyDuration.msgType = msgTypes.danger;
                            return;
                        }

                        //Save data for charting.
                        activityLogChartMaxAndAvgMonthlyDuration.rawData = j.Data || [];
                        activityLogChartMaxAndAvgMonthlyDuration.msg = "";

                        //Build the chart.
                        activityLogChartMaxAndAvgMonthlyDuration.buildChart();

                        return;
                    })
                    .catch(function (err) {
                        console.log("fetch() error: >>", err, "<<");
                        activityLogChartMaxAndAvgMonthlyDuration.msg = 'An unknown error occured. Please try again.';
                        activityLogChartMaxAndAvgMonthlyDuration.msgType = msgTypes.danger;
                        return;
                    });

                return;
            },

            //buildChart takes the raw data retrieved in getData() and builds and 
            //shows the chart.
            buildChart: function () {
                //Where chart will be shown.
                const elemID: string = 'activity-max-avg-duration-chart';
                var ctx: HTMLElement = document.getElementById(elemID);

                //Get the data points to chart.
                let yAxisYearMonths = [];
                let avg = [];
                let max = [];
                for (let r of (this.rawData as activityMaxAvgDuration[])) {
                    let yearMonth = r.Year + "-" + r.Month;
                    yAxisYearMonths.push(yearMonth);

                    avg.push(r.AverageTimeDuration);
                    max.push(r.MaxTimeDuration);
                }

                //Set chart options.
                let ops = {
                    legend: {
                        display: true, //don't show legend since we will describe data ourselves; legend is a bit messy
                    },
                    scales: {
                        leftYAxis: {
                            type: "linear",
                            position: "left",
                            ticks: {
                                min: 0,
                            }
                        },
                        rightYAxis: {
                            type: "linear",
                            position: "right",
                            ticks: {
                                min: 0,
                            },
                            grid: {
                                display: false,
                                displayOnChartArea: false,
                            }
                        },
    
                    },
                    animation: {
                        duration: 0, //don't animate chart when it is shown for better performance
                    },
                };

                //Generate the chart.
                //@ts-ignore cannot find name Chart
                reportActivityMaxAvgDurationChart = new Chart(ctx, {
                    type: 'line',
                    data: {
                        labels: yAxisYearMonths,
                        datasets: [
                            {
                                label: "Avg. (ms, left)",
                                borderColor: "#007bff",          //same as bootstrap primary
                                data: avg,
                                borderWidth: 2,                  //line thickness
                                yAxisID: "leftYAxis",
                                fill: false,              //no grey under lines, just show lines, more clear for users
                                lineTension: 0,                  //same as everwhere else in app
                            },
                            {
                                label: "Max. (ms, right)",
                                borderColor: "#17a2b8",          //same as bootstrap info
                                borderWidth: 2,                  //line thickness
                                borderDash: [5, 5],              //so that lines can be be differentiated to color blind people or when printed in black/white
                                data: max,
                                yAxisID: "rightYAxis",
                                fill: false,
                                lineTension: 0,
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
