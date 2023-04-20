/**
 * user-logins.ts
 * This retrieves the list of user logins.
 */

/// <reference path="common.ts" />
/// <reference path="fetch.ts" />
/// <reference path="types.ts" />

if (document.getElementById("userLogins")) {
    //@ts-ignore cannot find name Vue
    var userLogins = new Vue({
        name: 'userLogins',
        delimiters: ['[[', ']]'],
        el: '#userLogins',
        data: {
            //Filters.
            userID: 0, //specific user.
            rows: 50, //limit rows returned, not all rows need to be returned.

            //List of returned logins to build GUI with.
            logins: [],
            loginsRetrieved: false,

            //Data used to build filters.
            users: [] as user[],
            usersRetrieved: false,

            //Errors.
            msg: "",
            msgType: "",
            submitting: false,

            //Endpoints (for API requests, not for filters).
            urls: {
                getUsers: "/api/users/",
                getLogins: "/api/user-logins/latest/",
            },
        },
        methods: {
            //getUsers gets the list of users the list of logins can be filtered
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
                            userLogins.msg = err;
                            userLogins.msgType = msgTypes.danger;
                            return;
                        }

                        userLogins.users = j.Data || [];
                        userLogins.usersRetrieved = true;
                        return;
                    })
                    .catch(function (err) {
                        console.log("fetch() error: >>", err, "<<");
                        userLogins.msg = 'An unknown error occured. Please try again.';
                        userLogins.msgType = msgTypes.danger;
                        return;
                    });

                return;
            },

            //getLogins gets the list of the latest user logins filtered by the
            //filters.
            getLogins: function () {
                //Show loading message.
                this.msg = "Getting user login history...";
                this.msgType = msgTypes.primary;

                //Get logins.
                let data: Object = {
                    userID: this.userID,
                    rows: this.rows,
                };
                fetch(get(this.urls.getLogins, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then(function (j) {
                        //check if response is an error from the server
                        let err: string = handleAPIErrors(j);
                        if (err !== '') {
                            userLogins.msg = err;
                            userLogins.msgType = msgTypes.danger;
                            return;
                        }

                        userLogins.logins = j.Data || [];
                        userLogins.loginsRetrieved = true;

                        userLogins.msg = "";
                        return;
                    })
                    .catch(function (err) {
                        console.log("fetch() error: >>", err, "<<");
                        userLogins.msg = 'An unknown error occured. Please try again.';
                        userLogins.msgType = msgTypes.danger;
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

            return;
        }
    });
}
