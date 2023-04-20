/**
 * tools.ts
 * The code in this is used on the administrative tools page. This page
 * is typically hidden from the GUI just because it is rarely used and performs
 * some low level stuff in the db.
 */

/// <reference path="common.ts" />
/// <reference path="fetch.ts" />
/// <reference path="types.ts" />

if (document.getElementById("toolsClearActivityLog")) {
    //toolsClearActivityLog is used to clear the activity log of old data. This can 
    //help shrink the size of the db by clearing data from the activity log table 
    //which gets big pretty quick.
    //@ts-ignore cannot find name Vue
    var toolsClearActivityLog = new Vue({
        name: 'toolsClearActivityLog',
        delimiters: ['[[', ']]'],
        el: '#toolsClearActivityLog',
        data: {
            priorToDate: '', //yyyy-mm-dd
            msg: '',
            msgType: '',
            submitting: false,
        },
        methods: {
            clear: function () {
                //validate
                if (this.priorToDate === "") {
                    this.msg = "Invalid date provided.";
                    this.msgType = msgTypes.danger;
                    return;
                }

                //validation ok
                this.msg = 'Working...  This can take a while.';
                this.msgType = msgTypes.primary;
                this.submitting = true;

                //perform api call
                let data: Object = {
                    priorToDate: this.priorToDate
                };
                const url: string = "/api/activity-log/clear/";
                fetch(post(url, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then(function (j) {
                        //check if response is an error from the server
                        let err: string = handleAPIErrors(j);
                        if (err !== '') {
                            toolsClearActivityLog.msg = err;
                            toolsClearActivityLog.msgType = msgTypes.danger;
                            toolsClearActivityLog.submitting = false;
                            return;
                        }

                        toolsClearActivityLog.msg = "Done! Rows deleted: " + j.Data;
                        toolsClearActivityLog.msgType = msgTypes.success;
                        setTimeout(function () {
                            toolsClearActivityLog.msg = '';
                            toolsClearActivityLog.msgType = '';
                        }, defaultTimeout * 3);

                        toolsClearActivityLog.submitting = false;
                        return;
                    })
                    .catch(function (err) {
                        console.log("fetch() error: >>", err, "<<");
                        toolsClearActivityLog.msg = 'An unknown error occured. Please try again.';
                        toolsClearActivityLog.msgType = msgTypes.danger;
                        toolsClearActivityLog.submitting = false;
                        return;
                    });

                return;
            },
        },
    });
}

if (document.getElementById("toolsClearLogins")) {
    //toolsClearLogins is used to clear old user login history from the the database.
    //This can help shrink the size of the db by clearing data from the user logins
    //table which can get big if there are a lot of users, a short session timeout,
    //and/or both.
    //@ts-ignore cannot find name Vue
    var toolsClearLogins = new Vue({
        name: 'toolsClearLogins',
        delimiters: ['[[', ']]'],
        el: '#toolsClearLogins',
        data: {
            priorToDate: '', //yyyy-mm-dd
            msg: '',
            msgType: '',
            submitting: false,
        },
        methods: {
            clear: function () {
                //validate
                if (this.priorToDate === "") {
                    this.msg = "Invalid date provided.";
                    this.msgType = msgTypes.danger;
                    return;
                }

                //validation ok
                this.msg = 'Working...  This can take a while.';
                this.msgType = msgTypes.primary;
                this.submitting = true;

                //perform api call
                let data: Object = {
                    priorToDate: this.priorToDate
                };
                const url: string = "/api/users/login-history/clear/";
                fetch(post(url, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then(function (j) {
                        //check if response is an error from the server
                        let err: string = handleAPIErrors(j);
                        if (err !== '') {
                            toolsClearLogins.msg = err;
                            toolsClearLogins.msgType = msgTypes.danger;
                            toolsClearLogins.submitting = false;
                            return;
                        }

                        toolsClearLogins.msg = "Done! Rows deleted: " + j.Data;
                        toolsClearLogins.msgType = msgTypes.success;
                        setTimeout(function () {
                            toolsClearLogins.msg = '';
                            toolsClearLogins.msgType = '';
                        }, defaultTimeout * 3);

                        toolsClearLogins.submitting = false;
                        return;
                    })
                    .catch(function (err) {
                        console.log("fetch() error: >>", err, "<<");
                        toolsClearLogins.msg = 'An unknown error occured. Please try again.';
                        toolsClearLogins.msgType = msgTypes.danger;
                        toolsClearLogins.submitting = false;
                        return;
                    });

                return;
            },
        },
    });
}