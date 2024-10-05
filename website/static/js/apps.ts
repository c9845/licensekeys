/**
 * apps.ts
 * This file handles adding, viewing, and editing the apps you want to create license keys 
 * for. This is tied into defining/viewing key pairs and custom fields.
 */

/// <reference path="common.ts" />
/// <reference path="fetch.ts" />
/// <reference path="types.ts" />

if (document.getElementById("manageApps")) {
    //@ts-ignore cannot find name Vue
    var manageApps = new Vue({
        name: 'manageApps',
        delimiters: ['[[', ']]'],
        el: '#manageApps',
        data: {
            //handle gui states:
            // - page just loaded and no app selected: show lookup select inputs (uses addingNew & appData.ID)
            // - user wants to add app: show add/edit inputs.
            // - user chose a app: show lookup select inputs & add/edit inputs. (uses addingNew & appData.ID)
            addingNew: false,

            //loading apps from api call
            apps: [] as app[],
            appsRetrieved: false, //set to true once api call is complete, whether or not any apps were found. used to show loading message in select.
            msgLoad: '',
            msgLoadType: '',

            //options to choose from when adding/editing
            fileFormats: fileFormats,

            //defaults from list of options
            defaultFileFormat: fileFormatYAML,

            //app being added/edited
            appSelectedID: 0, //populated when an app is chosen from select menu
            appData: {} as app,

            //errors when adding or saving edits
            submitting: false,
            msgSave: '',
            msgSaveType: '',

            //endpoints
            urls: {
                get: "/api/apps/",
                add: "/api/apps/add/",
                update: "/api/apps/update/",
            },
        },
        computed: {
            //addEditCardTitle sets the title of the card used for adding or editing an app. We
            //need to set the title programmatically since the same card is used for adding and
            //editing.
            addEditCardTitle: function () {
                if (this.addingNew) {
                    return "Add App";
                }

                return "Edit App";
            },
        },
        methods: {
            //setField saves a chosen radio toggle's value to the Vue object
            setField: function (fieldName: string, value: boolean) {
                this.appData[fieldName] = value;
                return;
            },

            //resetForm sets the add form back to a clean state.
            //This is called in setUIState.
            resetForm: function () {
                this.appData = {
                    Name: "",
                    DaysToExpiration: 365,
                    FileFormat: this.defaultFileFormat,
                    DownloadFilename: "",
                    ShowLicenseID: true,
                    ShowAppName: true,
                    Active: true,
                } as app;

                //have to set toggles to match back to basics data
                //@ts-ignore Vue doesn't exist
                Vue.nextTick(function () {
                    setToggle('ShowLicenseID', true);
                    setToggle('ShowAppName', true);
                    setToggle('Active', true);
                });

                //no sense in having something selected if we are showing the add form or clearing the add form
                this.appSelectedID = 0;

                //remove any messages
                this.msgSave = '';
                this.msgSaveType = '';

                return;
            },

            //setUIState handles setting the gui state to the "add" or "edit/view" states based
            //upon user action (clicking add button, clicking lookup button). This also resets 
            //any inputs to empty/default values as needed.
            setUIState: function () {
                //User clicked on "lookup" button, user wants to see "lookup" UI. Don't clear 
                //inputs in case user clicked lookup button by mistake and is going to go back 
                //to adding. Plus, if user chooses aan app to lookup, the input values will be
                //reset as needed anyway.
                if (this.addingNew) {
                    this.addingNew = false;
                    return;
                }

                //User clicked on "add" button, user wants to see "add" form. Clear any existing
                //data so that inputs are blank.
                this.addingNew = true;
                this.resetForm();

                //Clear related card's data. Since we are adding a new app the other cards
                //should be blank.
                this.deselectApp()

                return;
            },

            //getApps gets the list of apps that have been defined.
            getApps: function () {
                let data: Object = {};
                fetch(get(this.urls.get, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then(function (j) {
                        //check if response is an error from the server
                        let err: string = handleAPIErrors(j);
                        if (err !== '') {
                            manageApps.msgLoad = err;
                            manageApps.msgLoadType = msgTypes.danger;
                            return;
                        }

                        //save data to display in gui
                        manageApps.apps = j.Data || [];
                        manageApps.appsRetrieved = true;

                        return;
                    })
                    .catch(function (err) {
                        console.log("fetch() error: >>", err, "<<");
                        manageApps.msgLoad = 'An unknown error occured. Please try again.';
                        manageApps.msgLoadType = msgTypes.danger;
                        return;
                    });

                return;
            },

            //showApp populates the "lookup" gui with data about the chosen app and is called
            //when an app is chosen from the list of apps. This also sets the chosen app's 
            //ID in other vue objects that manage other key pairs and custom fields. These vue 
            //objects will handle getting any data they need from the api (ex.: key pairs for
            //this app). This is also used to set the ui after an app is added.
            //
            //This is really just needed to set the correct toggles and pass data to other vue
            //objects. The setting of input values is done via v-model once the chosen app's
            //data is "moved" to "this.appData".
            showApp: function () {
                //make sure an app was selected
                if (this.appSelectedID < 1) {
                    return;
                }

                //get data for the chosen app
                for (let a of (this.apps as app[])) {
                    if (a.ID !== this.appSelectedID) {
                        continue;
                    }

                    //set this app which will show its data in the gui
                    this.appData = a;

                    //@ts-ignore Vue not found
                    Vue.nextTick(function () {
                        setToggle('ShowLicenseID', a.ShowLicenseID);
                        setToggle('ShowAppName', a.ShowAppName);
                        setToggle('Active', a.Active);
                    });

                    //set data in other vue objects as needed
                    this.setAppInOtherVueObjects(this.appSelectedID);
                    return;

                } //end for, loop through apps finding data for the chosen app

                return;
            },

            //setAppInOtherVueObjects sets the chosen app in vue objects that handle other 
            //parts of the gui. This func is called from within showApp.  
            setAppInOtherVueObjects: function (appID: number) {
                listKeyPairs.setAppID(appID);
                modalKeyPair.setAppID(appID);

                listCustomFieldsDefined.setAppID(appID);
                modalCustomFieldDefined.setAppID(appID);

                return;
            },

            //deselectApp resets the other related cards to a default state as if no app is
            //chosen. This is used when toggling from viewing an app to adding a new app to
            //remove the old app's data from the GUI.
            deselectApp: function () {
                this.setAppInOtherVueObjects(0)
                return;
            },

            //setDefaultDownloadFilename sets the download filename to a modified version
            //of the app's name, as long as the download filename is blank. This is used
            //to provide a default value so users don't have to come up with a filename
            //themselves.
            setDefaultDownloadFilename: function () {
                if (this.appData.Name.trim() === "") {
                    return;
                }
                // if (this.appData.DownloadFilename.trim() !== "") {
                //     return;
                // }

                let withoutSpaces: string = this.appData.Name.trim().replace(/ /g, "_");
                this.appData.DownloadFilename = withoutSpaces.toLowerCase() + "." + this.appData.FileFormat;
                return;
            },

            //addOrUpdate handles common validation before calling the correct function to
            //complete the api call.
            addOrUpdate: function () {
                //validation
                this.msgSaveType = msgTypes.danger;

                if (this.appData.Name.trim() === "") {
                    this.msgSave = "You must provide the name for your app.";
                    return;
                }
                if (this.appData.DaysToExpiration < 0) {
                    this.msgSave = "The default license period cannot be less than 0 days.";
                    return
                }
                if (!this.fileFormats.includes(this.appData.FileFormat.trim())) {
                    this.msgSave = "Please choose a file format from the provided options.";
                    return;
                }
                if (this.appData.DownloadFilename.trim() === "") {
                    this.msgSave = "You must provide the filename your licenses for this app will be downloaded as.";
                    return;
                }

                //handling of duplicate app names will be handled server side only.

                //call correct function
                if (this.appData.ID !== undefined) {
                    this.update();
                }
                else {
                    this.add();
                }

                return;
            },

            //add saves a new app. This is called from addOrUpdate().
            add: function () {
                //make sure data isn't already being submitted
                if (this.submitting) {
                    console.log("already submitting...");
                    return;
                }

                //validation ok
                this.msgSave = "Adding...";
                this.msgSaveType = msgTypes.primary;
                this.submitting = true;

                //perform api call
                let data: Object = {
                    data: JSON.stringify(this.appData),
                };
                fetch(post(this.urls.add, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then(function (j) {
                        //check if response is an error from the server
                        let err: string = handleAPIErrors(j);
                        if (err !== '') {
                            manageApps.msgSave = err;
                            manageApps.msgSaveType = msgTypes.danger;
                            manageApps.submitting = false;
                            return;
                        }

                        //Refresh the list of apps. We could just append this app to the list of
                        //apps we had already retrieved, but then we would need to deal with correct
                        //alphabetical order and we would be missing fields such as createdbyuserid
                        //or datetimecreated that we might want.
                        manageApps.getApps();

                        //Show success message for a few seconds. After success is chosen, the 
                        //GUI will populate with the new app's data.
                        manageApps.msgSave = "Added! Make sure you add the necessary key pairs and custom fields!";
                        manageApps.msgSaveType = msgTypes.success;
                        setTimeout(function () {
                            manageApps.appSelectedID = j.Data;  //"select" this app
                            manageApps.showApp();               //update the GUI for this app's data.
                            manageApps.setUIState();            //toggle back to "view/edit" mode.
                            manageApps.msgSave = '';
                            manageApps.msgLoadType = '';
                            manageApps.submitting = false;
                        }, defaultTimeout);

                        return;
                    })
                    .catch(function (err) {
                        console.log("fetch() error: >>", err, "<<");
                        manageApps.msgSave = 'An unknown error occured. Please try again.';
                        manageApps.msgSaveType = msgTypes.danger;
                        manageApps.submitting = false;
                        return;
                    });

                return;
            },

            //update saves changes to an existing app. This is called from addOrUpdate().
            update: function () {
                //make sure data isn't already being submitted
                if (this.submitting) {
                    console.log("already submitting...");
                    return;
                }

                //Make sure we know what app we are updating.
                if (isNaN(this.appData.ID) || this.appData.ID === '' || this.appData.ID < 1) {
                    this.msgSave = "Could not determine which app you are trying to update. Please refresh the page and try again.";
                    this.msgSaveType = msgTypes.danger;
                    return;
                }

                //validation ok
                this.msgSave = "Saving...";
                this.msgSaveType = msgTypes.primary;
                this.submitting = true;

                //perform api call
                let data: Object = {
                    data: JSON.stringify(this.appData),
                };
                fetch(post(this.urls.update, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then(function (j) {
                        //check if response is an error from the server
                        let err: string = handleAPIErrors(j);
                        if (err !== '') {
                            manageApps.msgSave = err;
                            manageApps.msgSaveType = msgTypes.danger;
                            manageApps.submitting = false;
                            return;
                        }

                        manageApps.msgSave = "Changes saved!";
                        manageApps.msgSaveType = msgTypes.success;
                        setTimeout(function () {
                            manageApps.msgSave = '';
                            manageApps.msgSaveType = '';
                            manageApps.submitting = false;
                        }, defaultTimeout);

                        return;
                    })
                    .catch(function (err) {
                        console.log("fetch() error: >>", err, "<<");
                        manageApps.msgSave = 'An unknown error occured. Please try again.';
                        manageApps.msgSaveType = msgTypes.danger;
                        manageApps.submitting = false;
                        return;
                    });

                return;

            },
        },
        mounted() {
            this.getApps();
            return;
        }
    })
}