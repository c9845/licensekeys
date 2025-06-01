/**
 * licenses.ts
 * 
 * This is used to show the list of licenses.
 */

import { createApp } from "vue";
import { msgTypes, apiBaseURL } from "./common";
import { get, handleRequestErrors, getJSON, handleAPIErrors } from "./fetch";
import { app, license } from "./types";
import { Tooltip } from "bootstrap";

if (document.getElementById("licenses")) {
    const licenses = createApp({
        name: 'licenses',

        compilerOptions: {
            delimiters: ['[[', ']]'],
        },

        data() {
            return {
                //Filters.
                apps: [] as app[],
                appsRetrieved: false,
                appSelectedID: 0,
                rowLimit: 20, //just a default value
                activeOnly: false, //not settable in gui as of now. not sure why we would want to filter by active or inactive licenses only.

                //Retrieved data.
                licenses: [] as license[],
                licensesRetrieved: false,

                //Form submission stuff.
                submitting: false,
                msg: '',
                msgType: '',

                //Endpoints.
                urls: {
                    getApps: apiBaseURL + "apps/",
                    getLicenses: apiBaseURL + "licenses/",
                }
            }
        },
        methods: {
            //getApps gets the list of apps for filtering licenses by.
            getApps: function () {
                let data: Object = {
                    activeOnly: true,
                };
                fetch(get(this.urls.getApps, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then((j) => {
                        //Check if response is an error from the server.
                        let err: string = handleAPIErrors(j);
                        if (err !== "") {
                            this.msgLoad = err;
                            this.msgLoadType = msgTypes.danger;
                            return;
                        }

                        this.apps = j.Data || [];
                        this.appsRetrieved = true;
                        return;
                    })
                    .catch((err) => {
                        console.log("fetch() error: >>", err, "<<");
                        this.msgLoad = "An unknown error occured. Please try again.";
                        this.msgLoadType = msgTypes.danger;
                        return;
                    });

                return;
            },

            //getLicenses looks up the list of licenses.
            getLicenses: function () {
                //Validate.
                if (this.appSelectedID < 0) {
                    this.appSelectedID = 0;
                }

                //Make API request.
                let data: Object = {
                    appID: this.appSelectedID,
                    limit: this.rowLimit,
                    activeOnly: this.activeOnly,
                };
                fetch(get(this.urls.getLicenses, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then((j) => {
                        //Check if response is an error from the server.
                        let err: string = handleAPIErrors(j);
                        if (err !== "") {
                            this.msgLoad = err;
                            this.msgLoadType = msgTypes.danger;
                            return;
                        }

                        this.licenses = j.Data || [];
                        this.licensesRetrieved = true;
                        return;
                    })
                    .catch((err) => {
                        console.log("fetch() error: >>", err, "<<");
                        this.msgLoad = "An unknown error occured. Please try again.";
                        this.msgLoadType = msgTypes.danger;
                        return;
                    });

                return;
            },
        },

        mounted() {
            //Look up apps the list of licenses can be filtered by.
            this.getApps();

            //Look up the list of licenses on page load.
            this.getLicenses();

            new Tooltip(document.body, {
                selector: "[data-bs-toggle='tooltip']",
            });

            return;
        }
    }).mount("#licenses");
}
