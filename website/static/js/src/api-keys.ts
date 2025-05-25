/**
 * api-keys.ts
 * 
 * The code in this is used on the manage API keys page. This is used to view, add, 
 * or revoke API keys.
 */

import { createApp } from "vue";
import { Modal } from "bootstrap";
import { msgTypes, apiBaseURL, defaultTimeout } from "./common";
import { get, post, handleRequestErrors, getJSON, handleAPIErrors } from "./fetch";

//Manage the list of API keys.
var manageAPIKeys: any; //must be "any", not "ComponentPublicInstance" to remove errors when calling functions (methods) of this Vue instance.
if (document.getElementById("manageAPIKeys")) {
    manageAPIKeys = createApp({
        name: 'manageAPIKeys',

        compilerOptions: {
            delimiters: ["[[", "]]"],
        },

        data() {
            return {
                //Handle GUI states:
                // - page just loaded and no API key selected: show lookup select inputs.
                // - user wants to add API key: show add/edit inputs.
                // - user chose an API key: show lookup select inputs & add/edit inputs.
                addingNew: false,

                //List of API keys.
                keys: [] as apiKey[],
                keysRetrieved: false,
                msgLoad: "",
                msgLoadType: "",

                //Single API key selected.
                apiKeySelectedID: 0,
                keyData: {
                    Description: "",
                    K: "",
                } as apiKey,

                //Form submission stuff.
                submitting: false,
                msgSave: "",
                msgSaveType: "",

                //Hide/show details.
                hidePermissionDescriptions: true,

                //Endpoints.
                urls: {
                    getKeys: apiBaseURL + "api-keys/",
                    generate: apiBaseURL + "api-keys/generate/",
                    update: apiBaseURL + "api-keys/update/",
                }
            }
        },
        computed: {
            //addEditCardTitle sets the text of the card used for adding or editing.
            //Since the card used for adding & editing is the same we want to show the 
            //correct card title to the user so they know what they are doing.
            addEditCardTitle: function () {
                if (this.addingNew) {
                    return "Add API Key";
                }

                return "Edit API Key";
            },
        },
        methods: {
            //getKeys looks up the list of existing, active, API keys.
            getKeys: function () {
                let data: Object = {};
                fetch(get(this.urls.getKeys, data))
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

                        this.keys = j.Data || [];
                        this.keysRetrieved = true;
                        return;
                    })
                    .catch((err) => {
                        console.log("fetch() error: >>", err, "<<");
                        this.msgLoad = "An unknown error occurred. Please try again.";
                        this.msgLoadType = msgTypes.danger;
                        return;
                    });

                return;
            },

            //createKey adds/generates a new API key.
            createKey: function () {
                //Validate.
                this.msgSaveType = msgTypes.danger;
                if (this.keyData.Description === "") {
                    this.msgSave = "You must provide a description for this API key so you can recognize what it is used for.";
                    return;
                }

                //Make API request.
                this.msgSave = "Creating...";
                this.msgSaveType = msgTypes.primary;
                this.submitting = true;

                let data: Object = {
                    data: JSON.stringify(this.keyData),
                };
                fetch(post(this.urls.generate, data))
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

                        //Refresh the list of API keys.
                        this.getKeys();

                        //Show success and the just-created API key.
                        this.msgSave = "API Key created";
                        this.msgSaveType = msgTypes.success;

                        setTimeout(() => {
                            this.resetForm();
                            this.msgSave = "";
                            this.msgSaveType = "";
                            this.submitting = false;

                            //Select this just created key.
                            this.apiKeySelectedID = j.Data.ID;

                            //Show the key for copying.
                            this.addingNew = false;
                            this.showAPIKey();
                        }, defaultTimeout);

                        return;
                    })
                    .catch((err) => {
                        console.log("fetch() error: >>", err, "<<");
                        this.msgSave = "An unknown error occurred. Please try again.";
                        this.msgSaveType = msgTypes.danger;
                        this.submitting = false;
                        return;
                    });
            },

            //update saves changes to an API key's description or permissions. You
            //cannot change the actual API key.
            update: function () {
                //Validation.
                if (isNaN(this.keyData.ID) || this.keyData.ID === "" || this.keyData.ID < 1) {
                    this.msgSave = "An error occurred. Please reload this page and try again.";
                    this.msgSaveType = msgTypes.danger;
                    return;
                }

                //Validate.
                this.msgSaveType = msgTypes.danger;
                if (this.keyData.ID < 1) {
                    this.msgSave = "Could not determine which API Key you want to update.";
                    return;
                }

                //Make API request.
                this.msgSave = "Saving...";
                this.msgSaveType = msgTypes.primary;
                this.submitting = true;

                let data: Object = {
                    data: JSON.stringify(this.keyData),
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

                        //Show success.
                        this.msgSave = "Changes saved!";
                        this.msgSaveType = msgTypes.success;

                        setTimeout(() => {
                            //Show success message.
                            this.msgSave = "";
                            this.msgSaveType = "";
                            this.submitting = false;
                        }, defaultTimeout);

                        return;
                    })
                    .catch((err) => {
                        console.log("fetch() error: >>", err, "<<");
                        this.msgSave = "An unknown error occurred. Please try again.";
                        this.msgSaveType = msgTypes.danger;
                        this.submitting = false;
                        return;
                    });
            },

            //createOrUpdate performs the correct action after performing common validation.
            createOrUpdate: function () {
                //Make sure data isn't already being submitted.
                if (this.submitting) {
                    console.log("already submitting...");
                    return;
                }

                if (this.addingNew) {
                    this.createKey();
                }
                else {
                    this.update();
                }

                return;
            },

            //updateAPIKeyList updates the list of API keys that can be chosen from after
            //an API key has been revoked. Inactive API keys are not displayed in the
            //GUI.
            //
            //This is called when an API key is revoked in the Vue object that handles
            //the revokation modal. 
            updateAPIKeyList: function () {
                this.getKeys();
                this.apiKeySelectedID = 0;
            },
        },
        mounted() {
            //Get list of existing API keys on page load.
            this.getKeys();
            return;
        }
    }).mount("#manageAPIKeys");
}

//Add, edit, and view an existing API key.
var modalRevokeAPIKey: any; //must be "any", not "ComponentPublicInstance" to remove errors when calling functions (methods) of this Vue instance.
if (document.getElementById("modal-revokeAPIKey")) {
    modalRevokeAPIKey = createApp({
        name: "modalRevokeAPIKey",

        compilerOptions: {
            delimiters: ["[[", "]]"],
        },

        data() {
            return {
                //apiKeyID is the API key that is being revoked.
                //
                //This is set in setKeyData() which is called from manageAPIKeys.showAPIKey()
                //when an API key is chosen.
                apiKeyID: 0,

                //Form submission stuff.
                submitting: false,
                msg: "",
                msgType: "",

                //Endpoints.
                urls: {
                    revoke: apiBaseURL + "api-keys/revoke/",
                },
            }
        },

        methods: {
            //setKeyData saves the provided API key's ID in this Vue instance.
            //
            //This is called any time an API key is chosen from the select menu in the
            //"lookup" GUI.
            setKeyData: function (apiKeyID: number) {
                this.apiKeyID = apiKeyID;
                return;
            },

            //revoke marks an API key as inactive. Inactive API keys cannot be used
            //and cannot be reactivated.
            revoke: function () {
                //Make sure data isn't already being saved.
                if (this.submitting) {
                    console.log("already submitting");
                    return;
                }

                //Validate.
                if (this.apiKeyID < 1) {
                    this.msg = "Could not determine which API key to revoke.";
                    this.msgType = msgTypes.danger;
                    return;
                }

                //Make API request.
                this.msg = "Revoking...";
                this.msgType = msgTypes.primary;
                this.submitting = true;

                let data: Object = {
                    id: this.apiKeyID,
                };
                fetch(post(this.urls.revoke, data))
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

                        //Show revoke message.
                        this.msg = "API Key was revoked successfully!"
                        this.msgType = msgTypes.success;

                        //Update the list of API keys so that the inactivated API key
                        //is removed.
                        manageAPIKeys.updateAPIKeyList();

                        //Automatically close the modal since there isn't anything else
                        //for the user to do.
                        setTimeout(() => {
                            let elem: HTMLElement = document.getElementById("modal-revokeAPIKey")!;
                            let modal = Modal.getInstance(elem)!;
                            modal.hide();

                            this.msg = "";
                            this.msgType = "";
                            this.submitting = false;
                        }, defaultTimeout);

                        return;
                    })
                    .catch((err) => {
                        console.log("fetch() error: >>", err, "<<");
                        this.msg = "An unknown error occurred. Please try again.";
                        this.msgType = msgTypes.danger;
                        this.submitting = false;
                        return;
                    });
            },
        }
    }).mount("#modal-revokeAPIKey");
}   