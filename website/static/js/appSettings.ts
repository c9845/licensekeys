/**
 * appSettings.ts
 * The code in this is used on the manage app settings.
 */

/// <reference path="common.ts" />
/// <reference path="fetch.ts" />
/// <reference path="types.ts" />

if (document.getElementById("manageApp")) {
    //manageApp handles the app settings page and saving changes to toggles.
    //@ts-ignore cannot find name Vue
    var manageApp = new Vue({
        name: 'manageApp',
        delimiters: ['[[', ']]'],
        el: '#manageApp',
        data: {
            //the app settings
            settings: {} as appSettings,

            //errors
            submitting: false,
            msg: '',
            msgType: '',

            //endpoints
            urls: {
                get: "/api/app-settings/",
                save: "/api/app-settings/update/",
            }
        },
        methods: {
            //getData gets the existing app settings from the db
            getData: function () {
                let data: Object = {};
                fetch(get(this.urls.get, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then(function (j) {
                        //check if response is an error from the server
                        let err: string = handleAPIErrors(j);
                        if (err !== '') {
                            manageApp.msg = err;
                            manageApp.msgType = msgTypes.danger;
                            return;
                        }

                        //save data to display in gui
                        manageApp.settings = j.Data;

                        //Set correct toggle state. Do this by iterating through list of 
                        //app settings retrieved from dataabse and setting each respective
                        //html btn group accordingly.
                        //Note the ignored fields; these are fields that don't have matching
                        //btn groups in the GUI.
                        //@ts-ignore Vue not found
                        Vue.nextTick(function () {
                            for (let key in manageApp.settings) {
                                if (key === "ID" || key === "DatetimeModified") {
                                    continue;
                                }
                                setToggle(key, manageApp.settings[key]);
                            }
                        });

                        return;
                    })
                    .catch(function (err) {
                        console.log("fetch() error: >>", err, "<<");
                        manageApp.msgSettings = 'An unknown error occured.  Please try again.';
                        manageApp.msgSettingsType = msgTypes.danger;
                        return;
                    });

                return;
            },

            //save saves changes made
            save: function () {
                if (this.submitting) {
                    console.log("already submitting...");
                    return;
                }

                //no real validation since things will just default to false on issues
                this.msg = "Saving...";
                this.msgType = msgTypes.primary;
                this.submitting = true;

                //perform api call
                let data: Object = {
                    data: JSON.stringify(this.settings),
                };
                fetch(post(this.urls.save, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then(function (j) {
                        //check if response is an error from the server
                        let err: string = handleAPIErrors(j);
                        if (err !== '') {
                            manageApp.msg = err;
                            manageApp.msgType = msgTypes.danger;
                            manageApp.submitting = false;
                            return;
                        }

                        //show success message
                        manageApp.msg = "Changes saved!";
                        manageApp.msgType = msgTypes.success;
                        setTimeout(function () {
                            manageApp.msg = '';
                            manageApp.msgType = '';
                        }, defaultTimeout);

                        manageApp.submitting = false;
                        return;
                    })
                    .catch(function (err) {
                        console.log("fetch() error: >>", err, "<<");
                        manageApp.msg = 'An unknown error occured.  Please try again.';
                        manageApp.msgType = msgTypes.danger;
                        manageApp.submitting = false;
                        return;
                    });

                return;
            },

            //setField saves a chosen radio toggle to the Vue object
            //this function is used because v-model doesn't work with bootstrap btn group radio
            setField: function (fieldName: string, value: boolean) {
                this.settings[fieldName] = value;
                return;
            },
        },
        mounted() {
            this.getData();
            return;
        }
    });
}
