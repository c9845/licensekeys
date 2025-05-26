/**
 * keyPairs.ts
 * 
 * This file handles adding, viewing, and deleting the key pairs assinged to an app 
 * for signing the license file data.
 */

import { createApp } from "vue";
import { msgTypes, apiBaseURL, defaultTimeout } from "./common";
import { get, post, handleRequestErrors, getJSON, handleAPIErrors } from "./fetch";

//Manage the list of key pairs.
export var listKeyPairs: any; //must be "any", not "ComponentPublicInstance" to remove errors when calling functions (methods) of this Vue instance.
if (document.getElementById("listKeyPairs")) {
    listKeyPairs = createApp({
        name: 'listKeyPairs',

        compilerOptions: {
            delimiters: ['[[', ']]'],
        },

        data() {
            return {
                //App to look up key pairs for. This is populated by setAppID.
                appSelectedID: 0,

                //List of key pairs.
                keyPairs: [] as keyPair[],
                keyPairsRetrieved: false,
                msgLoad: "",
                msgLoadType: "",

                //Handle GUI state. Collapse the card to take up less screen space.
                collapseUI: false,

                //Endpoints
                urls: {
                    get: apiBaseURL + "key-pairs/",
                }
            }
        },

        methods: {
            //setAppID sets the ID of the chosen app in this Vue object. 
            //
            //This is called from manageApps.setAppIDInOtherVueObjects() when an app 
            //is chosen from the list of defined apps. This then retrieves the list 
            //of key pairs for this app.
            setAppID: function (appID: number) {
                //Save the selected app's ID.
                this.appSelectedID = appID;

                //Handle adding a new app. No need to make API call when GUI is set
                //for adding a new app.
                if (appID === 0) {
                    this.keyPairs = [];
                    this.msgLoad = "";
                    return;
                }

                //Get the list of keypairs for the selected app.
                this.getKeyPairs();
                return;
            },

            //getKeyPairs gets the list of key pairs that have been defined for this 
            //app.
            getKeyPairs: function () {
                let data: Object = {
                    appID: this.appSelectedID,
                    activeOnly: true,
                };
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

                        this.keyPairs = j.Data || [];
                        this.keyPairsRetrieved = true;
                        return;
                    })
                    .catch((err) => {
                        console.log("fetch() error: >>", err, "<<");
                        this.msgLoad = "An unknown error occured. Please try again.";
                        this.msgLoadType = msgTypes.danger;
                        return;
                    });

                return;
            },

            //passToManageKeypairModal passes data about a chosen keypair to the modal
            //so we can display more info about the keypair than we can in a table.
            //This is done so that we don't have to pass along just the keypairID and
            //the perform a GET request to look up the keypair's data.
            //
            //This is called when clicking the "settings" button for a keypair or
            //clicking the "add" button to add a new key pair.
            //
            //When adding, "undefined" is passed as the keypair.
            //
            //Note that this does not open the modal, that is handled through bootstrap
            //data-toggle and data-target attributes.
            passToManageKeypairModal: function (kp: keyPair | undefined) {
                modalManageKeyPair.setKeypairInModal(this.appSelectedID, kp);
                return
            },
        }
    }).mount("#listKeyPairs");
}

//Handle displaying the full details of a key pair, and allowing management of it, or
//add a new key pair.
var modalManageKeyPair: any; //must be "any", not "ComponentPublicInstance" to remove errors when calling functions (methods) of this Vue instance.
if (document.getElementById("modal-manageKeyPair")) {
    modalManageKeyPair = createApp({
        name: 'modalManageKeyPair',

        compilerOptions: {
            delimiters: ['[[', ']]'],
        },

        data() {
            return {
                //App the keypair is for. We mostly need this for adding a keypair.
                // 
                //This is set in setKeypairInModal().
                appSelectedID: 0,

                //The keypair's data. Either for an existing keypair or for a new
                //keypair a user is adding.
                //
                //This is set in setKeypairInModal();
                keyPairData: {} as keyPair,

                //Options to choose from when adding.
                algorithmTypes: keyPairAlgoTypes,

                //Defaults for options.
                defaultAlgorithmType: keyPairAlgoED25519,

                //Show public key for copying. true upon button click to show public 
                //key in textarea for copying.
                showPublicKey: false,

                //Form submissing stuff.
                submitting: false,
                msgSave: '',
                msgSaveType: '',

                //Endpoints.
                urls: {
                    add: apiBaseURL + "key-pairs/add/",
                    delete: apiBaseURL + "key-pairs/delete/",
                    setDefault: apiBaseURL + "key-pairs/set-default/",
                }
            }
        },

        computed: {
            //adding is set to true when the user is adding/generating a key pair.
            // 
            //This is used to modify what the GUI displays (modal title and body text) 
            //to show correct helpful information based on adding or viewing a keypair.
            adding: function () {
                if (this.keyPairData.ID === undefined || this.keyPairData.ID < 1) {
                    return true;
                }

                return false;
            },

            //publicKeyNumLines returns the number of lines in the public key and is
            //used to set the "rows" attribute on the textarea so that the entire
            //public key is visible for ease of copying.
            publicKeyNumLines: function () {
                if (this.keyPairData.PublicKey === undefined || this.keyPairData.PublicKey.trim() === "") {
                    return 2; //default safe value
                }

                //Public key is a bunch of lines of text. To get number of lines,
                //split it by the newline character \n. Add 1 since the last line
                //in the public key doesn't end in a new line.
                return this.keyPairData.PublicKey.split('\n').length;
            },
        },

        methods: {
            //setKeypairInModal is used to populate the modal with data about a
            //keypair, or to set the modal to a clean state for adding a new keypair.
            // 
            //This is called from listKeyPairs.passToManageKeyPairModal() upon a user
            //clicking the "edit" or "add" buttons.
            setKeypairInModal: function (appID: number, kp: keyPair | undefined) {
                //Always reset.
                this.resetForm();

                //Always save the app ID.
                this.appSelectedID = appID;

                //User wants to add a new keypair.
                if (kp === undefined) {
                    return;
                }

                //User is viewing details of a keypair.
                this.keyPairData = kp;
                return;
            },

            //resetForm sets the modal back to a clean state for adding a new keypair.
            resetForm: function () {
                this.keyPairData = {
                    ID: 0,
                    DatetimeCreated: "", //wont be set, just to match type.
                    DatetimeModified: "", //" " "
                    CreatedByUserID: 0,  //" " "
                    Active: true,
                    AppID: this.appSelectedID,
                    Name: "",
                    AlgorithmType: this.defaultAlgorithmType,
                    PublicKey: "",
                    IsDefault: false,
                } as keyPair;

                this.showPublicKey = false;

                this.submitting = false;
                this.msgSave = "";
                this.msgSaveType = "";
                return;
            },

            //add saves a new keypair.
            add: function () {
                //Make sure data isn't already being submitted.
                if (this.submitting) {
                    console.log("already submitting...");
                    return;
                }

                //Validate.
                this.msgSaveType = msgTypes.danger;
                if (this.keyPairData.Name.trim() === "") {
                    this.msgSave = "You must provide a name for this key pair.";
                    return;
                }
                if (!this.algorithmTypes.includes(this.keyPairData.AlgorithmType)) {
                    this.msgSave = "Please choose an algorithm from the provided options";
                    return;
                }

                //Make API request.
                this.msgSave = "Generating key pair...";
                this.msgSaveType = msgTypes.primary;
                this.submitting = true;

                let data: Object = {
                    data: JSON.stringify(this.keyPairData),
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

                        //Update the local key pair data with data from the server. 
                        //This includes the keypair's ID and public key. This is done
                        //so user can copy the public key.
                        this.keyPairData = j.Data;

                        //Refresh the list of keypairs so that this new keypair is 
                        //shown.
                        listKeyPairs.getKeyPairs();

                        //Show success message.
                        this.msgSave = "Keypair added!";
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

            //remove marks a keypair as inactive. Inactive keypairs will no long show 
            //up in the list of keypairs or be available for use when creating a new 
            //license.
            //
            //Inactive keypairs cannot be reactivated since typically a keypair is
            //inactivated/removed for security reasons.
            remove: function () {
                //Make sure data isn't already being submitted.
                if (this.submitting) {
                    console.log("already submitting...");
                    return;
                }

                //Make sure we know what field we are deleting.
                if (isNaN(this.keyPairData.ID) || this.keyPairData.ID === '' || this.keyPairData.ID < 1) {
                    this.msgSave = "Could not determine which key pair you are trying to delete. Please refresh the page and try again.";
                    this.msgSaveType = msgTypes.danger;
                    return;
                }

                //Make API request.
                this.msgSave = "Deleting...";
                this.msgSaveType = msgTypes.primary;
                this.submitting = true;

                let data: Object = {
                    id: this.keyPairData.ID,
                };
                fetch(post(this.urls.delete, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then((j) => {
                        //Check if response is an error from the server.
                        let err: string = handleAPIErrors(j);
                        if (err !== "") {
                            this.msgSave = err;
                            this.msgSaveType = msgTypes.danger;
                            return;
                        }

                        //Refresh the list of keypairs in table. The modal will
                        //be closed by the data-dismiss on the button clicked that
                        //called this function.
                        listKeyPairs.getKeyPairs();
                        this.submitting = false;

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

            //setDefault marks this keypair as the default for the app. This can only 
            //be done for non-default keypairs.
            //
            //Setting a default keypair is useful so that when new licenses are 
            //created they will, by default, use this keypair. Typically this is used
            //when rotating keypairs.
            setDefault: function () {
                //Make sure data isn't already being submitted.
                if (this.submitting) {
                    console.log("already submitting...");
                    return;
                }

                //Make sure we know what field we are deleting.
                if (isNaN(this.keyPairData.ID) || this.keyPairData.ID === '' || this.keyPairData.ID < 1) {
                    this.msgSave = "Could not determine which key pair you want to set as default. Please refresh the page and try again.";
                    this.msgSaveType = msgTypes.danger;
                    return;
                }

                //Make API request.
                this.msgSave = "Setting default...";
                this.msgSaveType = msgTypes.primary;
                this.submitting = true;

                let data: Object = {
                    id: this.keyPairData.ID,
                };
                fetch(post(this.urls.setDefault, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then((j) => {
                        //Check if response is an error from the server.
                        let err: string = handleAPIErrors(j);
                        if (err !== "") {
                            this.msgSave = err;
                            this.msgSaveType = msgTypes.danger;
                            return;
                        }

                        //Set the "IsDefault" field for this keypair to update GUI.
                        //Also refresh the list of keypairs to show the default
                        //tag next to the correct keypair.
                        listKeyPairs.getKeyPairs();
                        this.keyPairData.IsDefault = true;
                        this.submitting = false;
                        this.msgSave = "";

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
            }
        }
    }).mount("#modal-manageKeyPair")
}