/**
 * tools.ts
 * 
 * The code in this is used on the administrative tools page.
 */

import { createApp } from "vue";
import { msgTypes, apiBaseURL, isValidDate, defaultTimeout } from "./common";
import { post, handleRequestErrors, getJSON, handleAPIErrors } from "./fetch";

if (document.getElementById("toolsClearActivityLog")) {
    const toolsClearActivityLog = createApp({
        name: "toolsClearActivityLog",

        compilerOptions: {
            delimiters: ["[[", "]]"],
        },

        data() {
            return {
                priorToDate: "", //yyyy-mm-dd

                //Form submission stuff.
                msg: "",
                msgType: "",
                submitting: false,

                //Endpoints.
                urls: {
                    clear: apiBaseURL + "activity-log/clear/",
                },
            }
        },

        methods: {
            //clear clears the activity log up to the chosen date.
            clear: function () {
                //Validate
                this.msgType = msgTypes.danger;
                if (this.priorToDate === "") {
                    this.msg = "Date was not provided.";
                    return;
                }
                if (!isValidDate(this.priorToDate)) {
                    this.msg = "Date is not valid."
                    return;
                }

                //Validation ok.
                this.msg = "Clearing activity log...  This can take a while.";
                this.msgType = msgTypes.primary;
                this.submitting = true;

                //Make API request.
                let data: Object = {
                    priorToDate: this.priorToDate
                };
                fetch(post(this.urls.clear, data))
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

                        //Show success message.
                        this.msg = "Done! Activities cleared: " + j.Data;
                        this.msgType = msgTypes.success;
                        setTimeout(() => {
                            this.msg = "";
                            this.msgType = "";
                        }, defaultTimeout * 3);

                        this.submitting = false;
                        return;
                    })
                    .catch((err) => {
                        console.log("fetch() error: >>", err, "<<");
                        this.msg = "An unknown error occurred. Please try again.";
                        this.msgType = msgTypes.danger;
                        this.submitting = false;
                        return;
                    });

                return;
            },
        },
    }).mount("#toolsClearActivityLog");
}

if (document.getElementById("toolsClearLogins")) {
    const toolsClearLogins = createApp({
        name: "toolsClearLogins",

        compilerOptions: {
            delimiters: ["[[", "]]"],
        },

        data() {
            return {
                priorToDate: "", //yyyy-mm-dd

                //Form submission stuff.
                msg: "",
                msgType: "",
                submitting: false,

                //Endpoints.
                urls: {
                    clear: apiBaseURL + "users/login-history/clear/",
                },
            }
        },

        methods: {
            //clear clears the user login history, aka user sessions, up to the chosen date.
            clear: function () {
                //Validate
                this.msgType = msgTypes.danger;
                if (this.priorToDate === "") {
                    this.msg = "Date was not provided.";
                    return;
                }
                if (!isValidDate(this.priorToDate)) {
                    this.msg = "Date is not valid."
                    return;
                }

                //Validation ok.
                this.msg = "Clearing login hisotry...  This can take a while.";
                this.msgType = msgTypes.primary;
                this.submitting = true;

                //Make API request.
                let data: Object = {
                    priorToDate: this.priorToDate
                };
                fetch(post(this.urls.clear, data))
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

                        //Show success message.
                        this.msg = "Done! Logins cleared: " + j.Data;
                        this.msgType = msgTypes.success;
                        setTimeout(() => {
                            this.msg = "";
                            this.msgType = "";
                        }, defaultTimeout * 3);

                        this.submitting = false;
                        return;
                    })
                    .catch((err) => {
                        console.log("fetch() error: >>", err, "<<");
                        this.msg = "An unknown error occurred. Please try again.";
                        this.msgType = msgTypes.danger;
                        this.submitting = false;
                        return;
                    });

                return;
            },
        },
    }).mount("#toolsClearLogins");
}