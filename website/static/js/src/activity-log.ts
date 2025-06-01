/**
 * activity-log.ts
 * 
 * This retrieves the list of the latest activities. This is the "main" activity log
 * page.
 */

import { createApp } from "vue";
import { Tooltip } from "bootstrap";
import { msgTypes, apiBaseURL, isValidDate } from "./common";
import { get, handleRequestErrors, getJSON, handleAPIErrors } from "./fetch";
import { activityLog, user, apiKey } from "./types";

if (document.getElementById("activityLog")) {
    const activityLog = createApp({
        name: 'activityLog',

        compilerOptions: {
            delimiters: ["[[", "]]"],
        },

        data() {
            return {
                //Filters.
                userID: 0, //specific user.
                apiKeyID: 0, //specific API key.
                endpoint: "", //page or endpoint URL.
                searchFor: "", //misc terms searched for in POST form values of a request.
                rows: 200, //limit rows returned, not all rows need to be returned.
                startDate: "",
                endDate: "",

                //Extras settings for displaying retrieved data.
                prettyPrintJSON: false,
                showReferrer: false,

                //List of returned activities to build GUI with.
                activities: [] as activityLog[],
                activitiesRetrieved: false,

                //Data used to build filters.
                users: [] as user[],
                usersRetrieved: false,
                apiKeys: [] as apiKey[],
                apiKeysRetrieved: false,
                endpoints: [] as string[],
                endpointsRetrieved: false,

                //Form submission stuff.
                msg: "",
                msgType: "",
                submitting: false,

                //Endpoints (for API requests, not for filters).
                urls: {
                    getLatest: apiBaseURL + "activity-log/latest/",
                    getEndpoints: apiBaseURL + "activity-log/latest/filter-by-endpoints/",
                    getUsers: apiBaseURL + "users/",
                    getAPIKeys: apiBaseURL + "api-keys/",
                },
            }
        },

        methods: {
            //getUsers gets the list of users the list of activities can be filtered by.
            getUsers: function () {
                let data: Object = {};
                fetch(get(this.urls.getUsers, data))
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

                        this.users = j.Data || [];
                        this.usersRetrieved = true;
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

            //getAPIKeys gets the list of API keys the list of activities can be 
            //filteredby.
            getAPIKeys: function () {
                let data: Object = {};
                fetch(get(this.urls.getAPIKeys, data))
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

                        this.apiKeys = j.Data || [];
                        this.apiKeysRetrieved = true;
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

            //getFilterByEndpoints gets the list of endpoints the list of activities
            //can be filtered by.
            getFilterByEndpoints: function () {
                let data: Object = {};
                fetch(get(this.urls.getEndpoints, data))
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

                        this.endpoints = j.Data || [];
                        this.endpointsRetrieved = true;
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

            //getActivities gets the list of the latest activities filtered by the
            //filters.
            getActivities: function () {
                //Validate.
                if (this.userID > 0 && this.apiKeyID > 0) {
                    this.msg = "Please choose a User or an API Key, not both.";
                    this.msgType = msgTypes.danger;
                }
                if (this.startDate !== "" && this.endDate === "") {
                    this.msgFilters = 'You must choose an End Date since you chose a Start Date.';
                    return;
                }
                if (this.endDate !== "" && this.startDate === "") {
                    this.msgFilters = 'You must choose an Start Date since you chose an End Date.';
                    return;
                }
                if (this.startDate !== "" && !isValidDate(this.startDate)) {
                    this.msgFilters = "The Start Date you provided is not valid.";
                    return
                }
                if (this.endDate !== "" && !isValidDate(this.endDate)) {
                    this.msgFilters = "The End Date you provided is not valid.";
                    return
                }

                //Make API request
                this.msg = "Getting activities...";
                this.msgType = msgTypes.primary;
                this.submitting = true;

                let data: Object = {
                    userID: this.userID,
                    apiKeyID: this.apiKeyID,
                    endpoint: this.endpoint,
                    searchFor: this.searchFor,
                    rows: this.rows,
                    startDate: this.startDate,
                    endDate: this.endDate,
                };
                fetch(get(this.urls.getLatest, data))
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

                        this.activities = j.Data || [];
                        this.activitiesRetrieved = true;
                        this.submitting = false;

                        this.msg = "";
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

        mounted() {
            //Get data on page load.
            this.getUsers();
            this.getAPIKeys();
            this.getFilterByEndpoints();

            //Populate page with latest activities.
            this.getActivities();

            //Enable tooltips.
            const tooltipTriggerList: any = document.querySelectorAll('[data-bs-toggle="tooltip"]');
            const tooltipList = [...tooltipTriggerList].map(tooltipTriggerEl => new Tooltip(tooltipTriggerEl));

            return;
        }
    }).mount("#activityLog");
}
