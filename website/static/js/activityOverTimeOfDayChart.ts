/**
 * activityOverTimeOfDayChart.ts
 * This builds the chart and hides/shows the help text for the chart that
 * shows the number of requests to the app over the time of day. This chart
 * is used for identifying when users are busy using the app.
 */

/// <reference path="common.ts" />

if (document.getElementById("activityOverTimeOfDayChart")) {
    //@ts-ignore cannot find name Vue
    var activityOverTimeOfDayChart = new Vue({
        name: 'activityOverTimeOfDayChart',
        delimiters: ['[[', ']]'],
        el: '#activityOverTimeOfDayChart',
        data: {
            showHelp: false,

            raw:        [], //raw json data from page load
            chartData:  undefined,
        },
        mounted() {
            //read the data in
            //This is the raw data retrieved from the db.
            //@ts-ignore rawChartData is set in activity-log-chart.html
            this.raw = rawChartData;

            //where chart will be shown
            const elemID: string = 'activity-over-time-of-day-chart';
            var ctx: HTMLElement = document.getElementById(elemID);

            //get the data points to chart
            let points = [];
            for(let r of this.raw) {
                let p: object = {
                    x: r.HoursDecimal,
                    y: r.Count,

                    //extra data for tooltips since using HoursDecimal is ugly in tooltips.
                    //These values represent the time "window" we grouped data into, 10 minute intervals.
                    hours:   r.Hour, 
                    minutes: r.Minute,
                };

                points.push(p);
            }

            //chart options
            let ops = {
                scales: {
                    x: {
                        type: 'linear',
                        ticks: {
                            min: 0,
                            max: 25,
                        }
                    },
                },
                animation: {
                    duration: 0, //don't animate chart when it is shown for better performance
                },
                plugins: {
                    legend: {
                        display: false, //don't show legend since we will describe data ourselves; legend is a bit messy
                    },
                    tooltip: {
                        callbacks: {
                            label: function(context) {
                                //Get index of what dataset this data point is from.
                                //This is the index of the dataset in chartDatasets or data.datasets.
                                let datasetIdx: number = context.datasetIndex;
    
                                //Get the data for this dataset.
                                //This gets the x/y values used to build the chart points.  
                                //This gets all the datapoints, not just the one being hovered on.
                                let dataset = activityOverTimeOfDayChart.chartData.data.datasets[datasetIdx].data;
                                
                                //Get the index of the point being hovered on.
                                //This ends up being the same index as the item in the dataset array.
                                let index: number = context.dataIndex;
                                
                                //Get the data used for the point being hovered on.
                                let point = dataset[index];
    
                                //look up the data for the point at this index
                                let num: number =   point.y;
                                let hours: string = point.hours;
                                let mins: string =  (parseInt(point.minutes) === 0) ? "00" : point.minutes;
    
                                return num + " @ " + hours + ":" + mins + " UTC";
                            }
                        }
                    }
                },
                
            };

            //generate the chart
            //@ts-ignore cannot find name Chart
            this.chartData = new Chart(ctx, {
                type: 'scatter',
                data: {
                    //labels: xAxis,
                    datasets: [
                        {
                            label:          "points",   //label for legend, but legend isn't used
                            data:           points,
                            borderColor:    '#007bff',      //primary, to match with app color scheme
                        },
                    ],
                },
                options: ops,
            });

            return;
        }
    });
}
