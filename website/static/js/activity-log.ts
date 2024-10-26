/**
 * activity-log.ts
 * This retrieves the list of the latest activities. This is the "main" activity log
 * page.
 */

/// <reference path="common.ts" />
/// <reference path="fetch.ts" />
/// <reference path="types.ts" />

if (document.getElementById("activityLog")) {
    //@ts-ignore cannot find name Vue
    var activityLog = new Vue({
        name: 'activityLog',
        delimiters: ['[[', ']]'],
        el: '#activityLog',
        data: {
            showHelp: false,

            //Filters.
            userID: 0, //specific user.
            apiKeyID: 0, //specific user.
            endpoint: "", //API endpoint URL.
            searchFor: "", //misc terms searched for in POST form values of a request.
            rows: 200, //limit rows returned, not all rows need to be returned.
            startDate: "",
            endDate: "",

            //Extras.
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

            //Errors.
            msg: "",
            msgType: "",
            submitting: false,

            //Endpoints (for API requests, not for filters).
            urls: {
                getLatest: "/api/activity-log/latest/",
                getEndpoints: "/api/activity-log/latest/filter-by-endpoints/",
                getUsers: "/api/users/",
                getAPIKeys: "/api/api-keys/",
            },
        },
        methods: {
            //getUsers gets the list of users the list of activities can be filtered
            //by.
            getUsers: function () {
                let data: Object = {};
                fetch(get(this.urls.getUsers, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then(function (j) {
                        //check if response is an error from the server
                        let err: string = handleAPIErrors(j);
                        if (err !== '') {
                            activityLog.msg = err;
                            activityLog.msgType = msgTypes.danger;
                            return;
                        }

                        activityLog.users = j.Data || [];
                        activityLog.usersRetrieved = true;
                        return;
                    })
                    .catch(function (err) {
                        console.log("fetch() error: >>", err, "<<");
                        activityLog.msg = 'An unknown error occurred. Please try again.';
                        activityLog.msgType = msgTypes.danger;
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
                    .then(function (j) {
                        //check if response is an error from the server
                        let err: string = handleAPIErrors(j);
                        if (err !== '') {
                            activityLog.msg = err;
                            activityLog.msgType = msgTypes.danger;
                            return;
                        }

                        activityLog.apiKeys = j.Data || [];
                        activityLog.apiKeysRetrieved = true;
                        return;
                    })
                    .catch(function (err) {
                        console.log("fetch() error: >>", err, "<<");
                        activityLog.msg = 'An unknown error occurred. Please try again.';
                        activityLog.msgType = msgTypes.danger;
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
                    .then(function (j) {
                        //check if response is an error from the server
                        let err: string = handleAPIErrors(j);
                        if (err !== '') {
                            activityLog.msg = err;
                            activityLog.msgType = msgTypes.danger;
                            return;
                        }

                        activityLog.endpoints = j.Data || [];
                        activityLog.endpointsRetrieved = true;
                        return;
                    })
                    .catch(function (err) {
                        console.log("fetch() error: >>", err, "<<");
                        activityLog.msg = 'An unknown error occurred. Please try again.';
                        activityLog.msgType = msgTypes.danger;
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

                //Show loading message.
                this.msg = "Getting activities...";
                this.msgType = msgTypes.primary;
                this.submitting = true;

                //Get activities.
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
                    .then(function (j) {
                        //check if response is an error from the server
                        let err: string = handleAPIErrors(j);
                        if (err !== '') {
                            activityLog.msg = err;
                            activityLog.msgType = msgTypes.danger;
                            activityLog.submitting = false;
                            return;
                        }

                        activityLog.activities = j.Data || [];
                        activityLog.activitiesRetrieved = true;
                        activityLog.submitting = false;

                        activityLog.msg = "";
                        return;
                    })
                    .catch(function (err) {
                        console.log("fetch() error: >>", err, "<<");
                        activityLog.msg = 'An unknown error occurred. Please try again.';
                        activityLog.msgType = msgTypes.danger;
                        activityLog.submitting = false;
                        return;
                    });

                return;
            },

            //setPrettyPrint saves a chosen radio toggle to the Vue object
            setPrettyPrint: function (value: boolean) {
                this.prettyPrintJSON = value;
                return;
            },

            //setShowReferrer saves a chosen radio toggle to the Vue object
            setShowReferrer: function (value: boolean) {
                this.showReferrer = value;
                return;
            },
        },
        mounted() {
            //Get filters.
            this.getUsers();
            this.getAPIKeys();
            this.getFilterByEndpoints();

            //Populate page with latest activities.
            this.getActivities();

            //Set default toggle value.
            setToggle("prettyPrintJSON", this.prettyPrintJSON);
            setToggle("showReferrer", this.showReferrer);

            return;
        }
    });
}
