/**
 * app-settings.ts
 * 
 * The code in this is used on the manage app settings.
 */

import { createApp } from "vue";
import { msgTypes, apiBaseURL, defaultTimeout, statementDescriptorMinLength, statementDescriptorMaxLength } from "./common";
import { get, post, handleRequestErrors, getJSON, handleAPIErrors } from "./fetch";

if (document.getElementById("manageAppSettings")) {
    const manageApp_App = createApp({
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
                        if (err !== '') {
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

                //Sanitize.
                this.appSettings.CustomerIDFormat = this.appSettings.CustomerIDFormat.trim();
                this.appSettings.CustomerIDRegex = this.appSettings.CustomerIDRegex.trim();
                this.appSettings.StatementDescriptor = this.appSettings.StatementDescriptor.trim();

                //Validate.
                let sdLen: number = this.appSettings.StatementDescriptor.length;
                if (sdLen < statementDescriptorMinLength || sdLen > statementDescriptorMaxLength) {
                    this.msg = "The StatementDescriptor must be between " + statementDescriptorMinLength + " and " + statementDescriptorMaxLength + " characters long. Yours is " + sdLen + " characters.";
                    return
                }
                if (this.appSettings.MinimumCharge < 1) {
                    this.msg = "The MinimumChargeCents must be at least 100.";
                    return
                }
                if (this.appSettings.AddCardLinkExpirationHours < 1) {
                    this.msg = "The AddCardLinkExpirationHours must be at least 1.";
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
                        if (err !== '') {
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
