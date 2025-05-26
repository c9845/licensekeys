/**
 * user-logins.ts
 * 
 * This retrieves the list of user logins, aka user sessions.
 */

import { createApp } from "vue";
import { Tooltip } from "bootstrap";
import { msgTypes, apiBaseURL, isValidDate } from "./common";
import { get, handleRequestErrors, getJSON, handleAPIErrors } from "./fetch";

if (document.getElementById("userLogins")) {
    const userLogins = createApp({
        name: "userLogins",

        compilerOptions: {
            delimiters: ["[[", "]]"],
        },

        data() {
            return {
                //Filters.
                userID: 0, //specific user.
                rows: 50, //limit rows returned, not all rows need to be returned.

                //List of returned logins to build GUI with.
                logins: [],
                loginsRetrieved: false,

                //Data used to build filters.
                users: [] as user[],
                usersRetrieved: false,

                //Form submission stuff.
                msg: "",
                msgType: "",
                submitting: false,

                //Endpoints.
                urls: {
                    getUsers: apiBaseURL + "users/",
                    getLogins: apiBaseURL + "user-logins/latest/",
                },
            }
        },
        methods: {
            //getUsers gets the list of users the list of logins can be filtered by.
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

                        this.logins = j.Data || [];
                        this.loginsRetrieved = true;
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

            //getLogins gets the list of the latest user logins filtered by the
            //filters.
            getLogins: function () {
                //Make API request.
                this.msg = "Getting user login history...";
                this.msgType = msgTypes.primary;

                let data: Object = {
                    userID: this.userID,
                    rows: this.rows,
                };
                fetch(get(this.urls.getLogins, data))
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

                        this.logins = j.Data || [];
                        this.loginsRetrieved = true;

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
            //Get filters.
            this.getUsers();

            //Populate page with latest logins.
            this.getLogins();

            //Enable tooltips.
            const tooltipTriggerList: any = document.querySelectorAll('[data-bs-toggle="tooltip"]');
            const tooltipList = [...tooltipTriggerList].map(tooltipTriggerEl => new Tooltip(tooltipTriggerEl));


            return;
        }
    }).mount("#userLogins");
}
