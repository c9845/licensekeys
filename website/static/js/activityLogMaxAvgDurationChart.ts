/**
 * activityLogMaxAvgDurationChart.ts
 * This builds the chart and hides/shows the help text for the chart that
 * shows the average and max latency for responding to requests to the app.
 * This chart is useful for identifying if the app is "getting slower" over
 * time by showing if latency (avg and max) is increasing.
 */

/// <reference path="common.ts" />

if (document.getElementById("activityLogMaxAvgDurationChart")) {
    //@ts-ignore cannot find name Vue
    var activityLogMaxAvgDurationChart = new Vue({
        name: 'activityLogMaxAvgDurationChart',
        delimiters: ['[[', ']]'],
        el: '#activityLogMaxAvgDurationChart',
        data: {
            showHelp: false,

            raw:        [], //raw json data from page load
            chartData:  undefined,
        },
        mounted() {
            //read the data in
            //This is the raw data retrieved from the db.
            //@ts-ignore rawChartData is set in activity-log-max-avg-duration-chart.html
            this.raw = rawChartData;

            //where chart will be shown
            const elemID: string = 'activity-max-avg-duration-chart';
            var ctx: HTMLElement = document.getElementById(elemID);

            //get the data points to chart
            let yAxisYearMonths =   [];
            let avg =               [];
            let max =               [];
            for(let r of this.raw) {
                let yearMonth = r.Year + "-" + r.Month;
                yAxisYearMonths.push(yearMonth);

                avg.push(r.AverageTimeDuration);
                max.push(r.MaxTimeDuration);
            }

            //chart options
            let ops = {
                legend: {
                    display: true, //don't show legend since we will describe data ourselves; legend is a bit messy
                },
                scales: {
                    leftYAxis: {
                        type:       "linear", 
                        position:   "left",
                        ticks: {
                            min: 0,
                        }
                    },
                    rightYAxis: {
                        type:       "linear", 
                        position:   "right",
                        ticks: {
                            min: 0,
                        },
                        grid: {
                            display:            false,
                            displayOnChartArea: false,
                        }
                    },
                    
                },
                animation: {
                    duration: 0, //don't animate chart when it is shown for better performance
                },
            };

            //generate the chart
            //@ts-ignore cannot find name Chart
            this.chartData = new Chart(ctx, {
                type: 'line',
                data: {
                    labels: yAxisYearMonths,
                    datasets: [
                        {
                            label:              "Avg. (ms, left)",
                            borderColor:        "#007bff",          //same as bootstrap primary
                            data:               avg,
                            borderWidth:        2,                  //line thickness
                            yAxisID:            "leftYAxis",
                            fill:               false,              //no grey under lines, just show lines, more clear for users
                            lineTension:        0,                  //same as everwhere else in app
                        },
                        {
                            label:              "Max. (ms, right)",
                            borderColor:        "#17a2b8",          //same as bootstrap info
                            borderWidth:        2,                  //line thickness
                            borderDash:         [5,5],              //so that lines can be be differentiated to color blind people or when printed in black/white
                            data:               max,
                            yAxisID:            "rightYAxis",
                            fill:               false,
                            lineTension:        0,
                        },
                    ]
                },
                options: ops,
            });

            return;
        }
    });
}
