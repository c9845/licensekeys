/**
 * activityLogRequestsDurationChart.ts
 * This builds the chart that shows the time the app took to repond
 * to the latest requests.
 */

/// <reference path="common.ts" />

if (document.getElementById("activityLogRequestsDurationChart")) {
    //@ts-ignore cannot find name Vue
    var activityLogRequestsDurationChart = new Vue({
        name: 'activityLogRequestsDurationChart',
        delimiters: ['[[', ']]'],
        el: '#activityLogRequestsDurationChart',
        data: {
            showHelp: false,

            raw:        [], //raw json data from page load
            chartData:  undefined,
        },
        mounted() {
            //read the data in
            //This is the raw data retrieved from the db.
            //@ts-ignore rawChartData is set in activity-log-chart-duration-latest-requests.html
            this.raw = rawChartData;

            //where chart will be shown
            const elemID: string = 'activity-request-duration-chart';
            var ctx: HTMLElement = document.getElementById(elemID);

            //get the data points to chart
            let xAxisLabels = [];
            let yAxisPoints = [];
            for (let r of this.raw) {
                xAxisLabels.push(r.DatetimeCreated);
                yAxisPoints.push(r.TimeDuration);
            }

            //chart options
            let ops = {
                legend: {
                    display: false, //don't show legend since we will describe data ourselves; legend is a bit messy
                },
                scales: {
                    y: {
                        type:       "linear", 
                        position:   "left",
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
                            title: function(context) {
                                let index: number = context[0].dataIndex;
                                let point = activityLogRequestsDurationChart.raw[index];
                                return point.Method + ": " + point.URL;
                            },
    
                            //this modifies the tooltip label to show the produced lot number or container code when a point is hovered
                            //Showing this extra info is helpful for figuring out what batch/raw material a point belongs to for diagnosing issues.
                            label: function(context) {
                                let index: number = context.dataIndex;
                                let point = activityLogRequestsDurationChart.raw[index];
                                return point.DatetimeCreated + " (" + point.TimeDuration + "ms)";
                            }
                        }
                    }
                }
            };

            //generate the chart
            //@ts-ignore cannot find name Chart
            this.chartData = new Chart(ctx, {
                type: 'bar',
                data: {
                    labels: xAxisLabels,
                    datasets: [
                        {
                            label:              "Duration (ms)",
                            backgroundColor:    "#007bff", //same as bootstrap primary
                            data:               yAxisPoints,
                            fill:               false,  //no grey under lines, just show lines, more clear for users
                            lineTension:        0,      //same as everwhere else in app
                        },
                    ]
                },
                options: ops,
            });

            return;
        }
    });
}
