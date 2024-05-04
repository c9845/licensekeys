/**
 * activity-log-chart-over-time-of-day.ts
 * 
 * This builds the chart that shows the activity within the app over the time of day
 * as a means of determine when the app is most being used. This is useful for 
 * identifying when your users are most active in the app.
 */

/// <reference path="common.ts" />

//chart is stored outside of Vue object to prevent "maximum call stack size exceeded".
let activityOverTimeOfDayChart = undefined;

interface activityOverTimeOfDay {
    //The number of activities that occurred within the time bracket/interval.
    Count: number,

    //Raw data for diagnostics.
    MinutesRaw: number, //00 - 60
    MinutesRounded: number, //00, 10, 20, 30 40, 50, 60
    HourRaw: number, //00 - 24

    //Calculated time brackets, taking into consideration minute bracket overlap.
    //AKA hour 2 minute 60 is the same as hour 3 minute 0.
    HourBracket: number,
    MinuteBracket: number,

    //Time as a decimal for ordering.
    HoursMinutesDecimal: number, //3:30 = 3.5
}

if (document.getElementById("activityLogChartOverTimeOfDay")) {
    //@ts-ignore cannot find name Vue
    var activityLogChartOverTimeOfDay = new Vue({
        name: 'activityLogChartOverTimeOfDay',
        delimiters: ['[[', ']]'],
        el: '#activityLogChartOverTimeOfDay',
        data: {
            showHelp: false,

            //Raw data to build chart with.
            rawData: [] as activityOverTimeOfDay[],

            //Errors.
            msg: "",
            msgType: "",

            //endpoints
            urls: {
                overTimeOfDay: "/api/activity-log/over-time-of-day/",
            },
        },
        methods: {
            //getData retrieves the data from the server.
            getData: function () {
                this.msg = "Retrieving data, this may take a while...";
                this.msgType = msgTypes.primary;

                let reqParams: Object = {};
                fetch(get(this.urls.overTimeOfDay, reqParams))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then(function (j) {
                        //check if response is an error from the server
                        let err: string = handleAPIErrors(j);
                        if (err !== '') {
                            activityLogChartOverTimeOfDay.msg = err;
                            activityLogChartOverTimeOfDay.msgType = msgTypes.danger;
                            return;
                        }

                        //Save data for charting.
                        activityLogChartOverTimeOfDay.rawData = j.Data || [];
                        activityLogChartOverTimeOfDay.msg = "";

                        //Build the chart.
                        activityLogChartOverTimeOfDay.buildChart();

                        return;
                    })
                    .catch(function (err) {
                        console.log("fetch() error: >>", err, "<<");
                        activityLogChartOverTimeOfDay.msg = 'An unknown error occurred. Please try again.';
                        activityLogChartOverTimeOfDay.msgType = msgTypes.danger;
                        return;
                    });

                return;
            },

            //buildChart takes the raw data retrieved in getData() and builds and 
            //shows the chart.
            buildChart: function () {
                //Where chart will be shown.
                const elemID: string = 'activity-over-time-of-day-chart';
                var ctx: HTMLElement = document.getElementById(elemID);

                //Get the data points to chart.
                let points = [];
                for (let r of (this.rawData as activityOverTimeOfDay[])) {
                    let p: object = {
                        x: r.HoursMinutesDecimal,
                        y: r.Count,

                        //Extra data for tooltips since using HoursDecimal is ugly in 
                        //tooltips. These values represent the time "window"/interval
                        //we grouped data into (default is 10 minute intervals).
                        hours: r.HourBracket,
                        minutes: r.MinuteBracket,
                    };

                    points.push(p);
                }

                //Set chart options.
                let ops = {
                    scales: {
                        x: {
                            type: 'linear',
                            ticks: {
                                min: 0,
                                max: 24,
                                stepSize: 1,
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
                                label: function (context) {
                                    //Handle single digit hours and minutes, always
                                    //use two digits so tooltips are more consistent.
                                    let hours: string = (context.raw.hours).toString();
                                    if (hours.length === 1) {
                                        hours = "0" + hours;
                                    }

                                    let minutes: string = (context.raw.minutes).toString();
                                    if (minutes.length === 1) {
                                        minutes = "0" + minutes;
                                    }

                                    //Build tooltip content.
                                    return context.raw.y + " @ " + hours + ":" + minutes;
                                }
                            }
                        }
                    },
                };

                //Generate the chart.
                //@ts-ignore cannot find name Chart
                activityOverTimeOfDayChart = new Chart(ctx, {
                    type: 'scatter',
                    data: {
                        //labels: xAxis,
                        datasets: [
                            {
                                label: "points",        //label for legend, but legend isn't used
                                data: points,
                                borderColor: '#007bff', //primary, to match with app color scheme
                            },
                        ],
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
