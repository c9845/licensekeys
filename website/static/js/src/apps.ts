/**
 * apps.ts
 * 
 * This file handles adding, viewing, and editing the apps you want to create license keys 
 * for. This is tied into defining/viewing key pairs and custom fields.
 */

import { createApp } from "vue";
import { msgTypes, apiBaseURL, defaultTimeout } from "./common";
import { get, post, handleRequestErrors, getJSON, handleAPIErrors } from "./fetch";
import { listKeyPairs } from "./key-pairs";
import { listCustomFieldsDefined } from "./custom-fields-defined";
import { app } from "./types";
import { Tooltip } from "bootstrap";

//Manage the list of apps you can create licenses for.
var manageApps: any; //must be "any", not "ComponentPublicInstance" to remove errors when calling functions (methods) of this Vue instance.
if (document.getElementById("manageApps")) {
    manageApps = createApp({
        name: 'manageApps',

        compilerOptions: {
            delimiters: ['[[', ']]'],
        },

        data() {
            return {
                //Handle GUI states:
                // - page just loaded and no app selected: show lookup select inputs.
                // - user wants to add app: show add/edit inputs.
                // - user chose a app: show lookup select inputs & add/edit inputs.
                addingNew: false,

                //List of apps.
                apps: [] as app[],
                appsRetrieved: false,
                msgLoad: "",
                msgLoadType: "",

                //Single app selected or being added.
                appSelectedID: 0, //populated when an app is chosen from select menu
                appData: {
                    Name: "",
                    DaysToExpiration: 365,
                    FileFormat: "json",
                    DownloadFilename: "",
                    ShowLicenseID: true,
                    ShowAppName: true,
                    Active: true, //New apps are always active, because why would you create a new app if they are inactive?
                } as app,

                //Form submission stuff.
                submitting: false,
                msgSave: "",
                msgSaveType: "",

                //Endpoints.
                urls: {
                    get: apiBaseURL + "apps/",
                    add: apiBaseURL + "apps/add/",
                    update: apiBaseURL + "apps/update/",
                },
            }
        },

        computed: {
            //addEditCardTitle sets the text of the card used for adding or editing.
            //Since the card used for adding & editing is the same we want to show the 
            //correct card title to the user so they know what they are doing.
            addEditCardTitle: function () {
                if (this.addingNew) {
                    return "Add App";
                }

                return "Edit App";
            },
        },

        methods: {
            //getApps gets the list of apps that have been defined.
            getApps: function () {
                let data: Object = {};
                fetch(get(this.urls.get, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then((j) => {
                        //Check if response is an error from the server.
                        let err: string = handleAPIErrors(j);
                        if (err !== "") {
                            this.msgLoad = err;
                            this.msgLoadType = msgTypes.danger;
                            return;
                        }

                        this.apps = j.Data || [];
                        this.appsRetrieved = true;
                        return;
                    })
                    .catch((err) => {
                        console.log("fetch() error: >>", err, "<<");
                        this.msgLoad = 'An unknown error occured. Please try again.';
                        this.msgLoadType = msgTypes.danger;
                        return;
                    });

                return;
            },

            //setState handles setting the GUI state to the "add" or "looup/edit" 
            //state.
            //
            //This is called when a user clicks the add or view buttons in the card
            //header or when a new app is saved.
            setState: function () {
                //User clicked on "lookup/edit" button, user wants to see "lookup" UI.
                if (this.addingNew) {
                    this.addingNew = !this.addingNew;
                    return;
                }

                //User clicked on "add" button, user wants to see "add" UI. Clear 
                //any existing user data so that the inputs are reset to a blank state
                //for saving of a new user.
                this.addingNew = !this.addingNew;
                this.resetForm();

                this.msgSave = "";
                this.msgSaveType = "";

                return;

                //Clear related card's data. Since we are adding a new app the other cards
                //should be blank.
                //TODO: fix or remove.
                this.deselectApp()

                return;
            },

            //resetForm sets the add form back to a clean state.
            //This is called in setState.
            resetForm: function () {
                this.appData = {
                    Name: "",
                    DaysToExpiration: 365,
                    FileFormat: "json",
                    DownloadFilename: "",
                    ShowLicenseID: true,
                    ShowAppName: true,
                    Active: true,
                } as app;

                this.appSelectedID = 0;

                return;
            },

            //showApp populates the "lookup" GUI with data about the app chosen from
            //the select menu.
            showApp: function () {
                //Make sure an app was selected.
                if (this.appSelectedID < 1) {
                    return;
                }

                //Get app's data drom the list of apps we retrieved.
                for (let a of (this.apps as app[])) {
                    if (a.ID !== this.appSelectedID) {
                        continue;
                    }

                    //Save the chosen app for displaying in the GUI.
                    this.appData = a;
                    break;
                }

                //Set the app in other Vue object.
                this.setAppIDInOtherVueObjects(this.appSelectedID);

                return;
            },

            //setAppIDInOtherVueObjects sets the chosen app in Vue objects that handle 
            //other parts of the GUI.
            // 
            //This func is called from showApp().
            setAppIDInOtherVueObjects: function (appID: number) {
                listKeyPairs.setAppID(appID);

                listCustomFieldsDefined.setAppID(appID);

                return;
            },

            //deselectApp resets the other related cards to a default state as if no app is
            //chosen. This is used when toggling from viewing an app to adding a new app to
            //remove the old app's data from the GUI.
            deselectApp: function () {
                this.setAppIDInOtherVueObjects(0)
                return;
            },

            //setDefaultDownloadFilename sets the download filename to a modified 
            //version of the app's name, as long as the download filename is blank. 
            //
            //This is used to provide a default value so users don't have to come up 
            //with a filename themselves.
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

            //addOrUpdate performs the correct action after performing common 
            //validation.
            addOrUpdate: function () {
                //Make sure data isn't already being submitted.
                if (this.submitting) {
                    console.log("already submitting...");
                    return;
                }

                //Validation.
                this.msgSaveType = msgTypes.danger;

                if (this.appData.Name.trim() === "") {
                    this.msgSave = "You must provide the name for your app.";
                    return;
                }
                if (this.appData.DaysToExpiration < 0) {
                    this.msgSave = "The default license period cannot be less than 0 days.";
                    return
                }
                if (this.appData.DownloadFilename.trim() === "") {
                    this.msgSave = "You must provide the filename your licenses for this app will be downloaded as.";
                    return;
                }

                //handling of duplicate app names will be handled server side only.

                //Perform correct task.
                if (this.appData.ID !== undefined) {
                    this.update();
                }
                else {
                    this.add();
                }

                return;
            },

            //add saves a new app.
            add: function () {
                //Validation.

                //Make API request.
                this.msgSave = "Adding...";
                this.msgSaveType = msgTypes.primary;
                this.submitting = true;

                let data: Object = {
                    data: JSON.stringify(this.appData),
                };
                fetch(post(this.urls.add, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then((j) => {
                        //Check if response is an error from the server.
                        let err: string = handleAPIErrors(j);
                        if (err !== "") {
                            this.msgSave = err;
                            this.msgSaveType = msgTypes.danger;
                            this.submitting = false;
                            return;
                        }

                        //Refresh the list of apps.
                        this.getApps();

                        //Show success and reset the form.
                        //
                        //The GUI will be populated with this app's data so that user
                        //can continue settings up this app (key pairs, custom fields).
                        this.msgSave = "App Added! Make sure you add the necessary key pairs and custom fields!";
                        this.msgSaveType = msgTypes.success;
                        setTimeout(() => {
                            //"select" this app.
                            this.appSelectedID = j.Data;
                            this.showApp();
                            this.setState();

                            this.msgSave = "";
                            this.msgLoadType = "";
                            this.submitting = false;
                        }, defaultTimeout);

                        return;
                    })
                    .catch((err) => {
                        console.log("fetch() error: >>", err, "<<");
                        this.msgSave = "An unknown error occured. Please try again.";
                        this.msgSaveType = msgTypes.danger;
                        this.submitting = false;
                        return;
                    });

                return;
            },

            //update saves changes to an existing app.
            update: function () {
                //Validation.
                if (isNaN(this.appData.ID) || this.appData.ID === '' || this.appData.ID < 1) {
                    this.msgSave = "Could not determine which app you are trying to update. Please refresh the page and try again.";
                    this.msgSaveType = msgTypes.danger;
                    return;
                }

                //Make API request.
                this.msgSave = "Saving...";
                this.msgSaveType = msgTypes.primary;
                this.submitting = true;

                let data: Object = {
                    data: JSON.stringify(this.appData),
                };
                fetch(post(this.urls.update, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then((j) => {
                        //Check if response is an error from the server.
                        let err: string = handleAPIErrors(j);
                        if (err !== "") {
                            this.msgSave = err;
                            this.msgSaveType = msgTypes.danger;
                            this.submitting = false;
                            return;
                        }

                        //Show success message.
                        this.msgSave = "Changes saved!";
                        this.msgSaveType = msgTypes.success;
                        setTimeout(() => {
                            this.msgSave = "";
                            this.msgSaveType = "";
                            this.submitting = false;
                        }, defaultTimeout);

                        return;
                    })
                    .catch((err) => {
                        console.log("fetch() error: >>", err, "<<");
                        this.msgSave = "An unknown error occured. Please try again.";
                        this.msgSaveType = msgTypes.danger;
                        this.submitting = false;
                        return;
                    });

                return;

            },
        },

        mounted() {
            //Load the apps that exist that a user can choose from.
            this.getApps();

            new Tooltip(document.body, {
                selector: "[data-bs-toggle='tooltip']",
            });

            return;
        },
    }).mount("#manageApps")
}