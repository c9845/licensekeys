/**
 * activity-log-chart-duration-by-endpoint.ts
 * 
 * This builds the table that shows the max/min/avg duration it took to respond to
 * each endpoint. This can be useful for identifying which endpoints take a long time
 * to respond.
 */

import { createApp } from "vue";
import { msgTypes, apiBaseURL } from "./common";
import { get, handleRequestErrors, getJSON, handleAPIErrors } from "./fetch";

interface durationByEndpoint {
    URL: string,
    EndpointHits: number,
    Method: string,
    AverageTimeDuration: number,
    MaxTimeDuration: number,
    MinTimeDuration: number,
}

if (document.getElementById("durationByEndpoint")) {
    const durationByEndpoint = createApp({
        name: 'durationByEndpoint',

        compilerOptions: {
            delimiters: ["[[", "]]"],
        },

        data() {
            return {
                //Show help, description of what chart is showing.
                showHelp: false,

                //Raw data to build chart with.
                rawData: [] as durationByEndpoint[],

                //Errors.
                msg: "",
                msgType: "",

                //Endpoints.
                urls: {
                    maxAvgDuration: apiBaseURL + "activity-log/duration-by-endpoint/",
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
                fetch(get(this.urls.maxAvgDuration, reqParams))
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

                        //Save data for building table.
                        this.rawData = j.Data || [];
                        this.msg = "";

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
        },
        mounted() {
            //Make request to get data, which will then build the table.
            this.getData();
            return;
        }
    }).mount("#durationByEndpoint");
}