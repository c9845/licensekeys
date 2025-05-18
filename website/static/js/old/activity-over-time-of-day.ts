/**
 * activity-over-time-of-day.ts
 * 
 * This builds the chart that shows the activity within the app over the time of day
 * as a means of determine when the app is most being used. This is useful for 
 * identifying when your users are most active in the app.
 */

import { createApp } from "vue";
import { Chart, ChartOptions } from "chart.js/auto"; //auto just makes importing easier.
import { msgTypes, apiBaseURL } from "./common";
import { get, handleRequestErrors, getJSON, handleAPIErrors } from "./fetch";

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

if (document.getElementById("activityOverTimeOfDay")) {
    const activityOverTimeOfDay = createApp({
        name: 'activityOverTimeOfDay',

        compilerOptions: {
            delimiters: ["[[", "]]"],
        },

        data() {
            return {
                //Show help, description of what chart is showing.
                showHelp: false,

                //Raw data to build chart with.
                rawData: [] as activityOverTimeOfDay[],

                //Errors.
                msg: "",
                msgType: "",

                //endpoints
                urls: {
                    overTimeOfDay: apiBaseURL + "activity-log/over-time-of-day/",
                },
            }
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

            //buildChart takes the raw data retrieved in getData() and builds and shows 
            //the chart.
            buildChart: function () {
                //Where chart will be shown.
                const elemID: string = "activity-over-time-of-day-chart";
                var ctx: HTMLElement = document.getElementById(elemID)!;

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
                let ops: ChartOptions = {
                    scales: {
                        x: {
                            type: "linear",
                            min: 0,
                            max: 24,
                            ticks: {
                                stepSize: 1,
                            }
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
                                label: function (context: any) {
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
                    }
                };

                //Generate the chart.
                //@ts-ignore cannot find name Chart
                activityOverTimeOfDayChart = new Chart(ctx, {
                    type: 'scatter',
                    data: {
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
    }).mount("#activityOverTimeOfDay");
}
