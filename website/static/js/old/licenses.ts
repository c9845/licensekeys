/**
 * licenses.ts
 * This is used to show the list of licenses.
 */

/// <reference path="common.ts" />
/// <reference path="fetch.ts" />
/// <reference path="types.ts" />

if (document.getElementById("licenses")) {
    //@ts-ignore cannot find name Vue
    var licenses = new Vue({
        name: 'licenses',
        delimiters: ['[[', ']]'],
        el: '#licenses',
        data: {
            //filters
            apps: [] as app[],
            appsRetrieved: false,
            appSelectedID: 0,
            rowLimit: 20, //just a default value
            activeOnly: false, //not settable in gui (yet)

            //retrieved data
            licenses: [] as license[],
            licensesRetrieved: false,

            submitting: false,
            msg: '',
            msgType: '',

            //endpoints
            urls: {
                getApps: "/api/apps/",
                getLicenses: "/api/licenses/",
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
                    .then(function (j) {
                        //check if response is an error from the server
                        let err: string = handleAPIErrors(j);
                        if (err !== '') {
                            licenses.msgLoad = err;
                            licenses.msgLoadType = msgTypes.danger;
                            return;
                        }

                        //save data to display in gui
                        licenses.apps = j.Data || [];
                        licenses.appsRetrieved = true;

                        return;
                    })
                    .catch(function (err) {
                        console.log("fetch() error: >>", err, "<<");
                        licenses.msgLoad = 'An unknown error occured. Please try again.';
                        licenses.msgLoadType = msgTypes.danger;
                        return;
                    });

                return;
            },

            //getLicenses looks up the list of licenses.
            getLicenses: function () {
                //validate
                if (this.appSelectedID < 0) {
                    this.appSelectedID = 0;
                }

                let data: Object = {
                    appID: this.appSelectedID,
                    limit: this.rowLimit,
                    activeOnly: this.activeOnly,
                };
                fetch(get(this.urls.getLicenses, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then(function (j) {
                        //check if response is an error from the server
                        let err: string = handleAPIErrors(j);
                        if (err !== '') {
                            licenses.msgLoad = err;
                            licenses.msgLoadType = msgTypes.danger;
                            return;
                        }

                        licenses.licenses = j.Data || [];
                        licenses.licensesRetrieved = true;
                        return;
                    })
                    .catch(function (err) {
                        console.log("fetch() error: >>", err, "<<");
                        licenses.msgLoad = 'An unknown error occured.  Please try again.';
                        licenses.msgLoadType = msgTypes.danger;
                        return;
                    });

                return;
            },
        },
        mounted() {
            //look up apps the list of licenses can be filtered by
            this.getApps();

            //look up the list of licenses on page load
            this.getLicenses();

            return;
        }
    });
}
