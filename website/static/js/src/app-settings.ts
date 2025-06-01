/**
 * app-settings.ts
 * 
 * The code in this is used on the manage app settings.
 */

import { createApp } from "vue";
import { msgTypes, apiBaseURL, defaultTimeout } from "./common";
import { get, post, handleRequestErrors, getJSON, handleAPIErrors } from "./fetch";
import { appSettings } from "./types";

if (document.getElementById("manageAppSettings")) {
    const manageAppSettings = createApp({
        name: 'manageApp',

        compilerOptions: {
            delimiters: ["[[", "]]"],
        },

        data() {
            return {
                //App settings. Retrieved via API call.
                appSettings: {} as appSettings,

                //Form submission stuff.
                submitting: false,
                msg: '',
                msgType: '',

                //Endpoints.
                urls: {
                    get: apiBaseURL + "app-settings/",
                    save: apiBaseURL + "app-settings/update/",
                }
            }
        },

        methods: {
            //getData gets the app settings from the db.
            getData: function () {
                let data: Object = {};
                fetch(get(this.urls.get, data))
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

                        //Save data to display in GUI.
                        this.appSettings = j.Data || {};
                        return;
                    })
                    .catch((err) => {
                        console.log("fetch() error: >>", err, "<<");
                        this.msgSettings = 'An unknown error occured. Please try again.';
                        this.msgSettingsType = msgTypes.danger;
                        return;
                    });

                return;
            },

            //update saves changes made to the app settings.
            update: function () {
                if (this.submitting) {
                    console.log("already submitting...");
                    return;
                }

                //Make API request.
                this.msg = "Saving...";
                this.msgType = msgTypes.primary;
                this.submitting = true;

                let data: Object = {
                    data: JSON.stringify(this.appSettings),
                };
                fetch(post(this.urls.save, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then((j) => {
                        //Check if response is an error from the server.
                        let err: string = handleAPIErrors(j);
                        if (err !== "") {
                            this.msg = err;
                            this.msgType = msgTypes.danger;
                            this.submitting = false;
                            return;
                        }

                        this.msg = "Changes saved!";
                        this.msgType = msgTypes.success;
                        setTimeout(() => {
                            this.msg = '';
                            this.msgType = '';
                            this.submitting = false;
                        }, defaultTimeout);

                        return;
                    })
                    .catch((err) => {
                        console.log("fetch() error: >>", err, "<<");
                        this.msg = 'An unknown error occured. Please try again.';
                        this.msgType = msgTypes.danger;
                        this.submitting = false;
                        return;
                    });

                return;
            },
        },

        mounted() {
            //Load data for app settings on page load.
            this.getData();
            return;
        }
    }).mount("#manageAppSettings");
}
