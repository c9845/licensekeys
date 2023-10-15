/**
 * api-keys.ts
 * 
 * The code in this is used on the manage API keys page. This is used to view, add, 
 * or revoke API keys.
 */

/// <reference path="common.ts" />
/// <reference path="fetch.ts" />
/// <reference path="types.ts" />

if (document.getElementById("manageAPIKeys")) {
    //manageAPIKeys handles the list of API keys.
    //@ts-ignore cannot find name Vue
    var manageAPIKeys = new Vue({
        name: 'manageAPIKeys',
        delimiters: ['[[', ']]'],
        el: '#manageAPIKeys',
        data: {
            //List of API keys.
            keys: [] as apiKey[],
            keysRetrieved: false,

            //errors
            submitting: false,
            msg: '',
            msgType: '',

            //endpoints
            urls: {
                getKeys: "/api/api-keys/",
            }
        },
        methods: {
            //getKeys gets the list of existing, active, API keys.
            getKeys: function () {
                let data: Object = {};
                fetch(get(this.urls.getKeys, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then(function (j) {
                        //check if response is an error from the server
                        let err: string = handleAPIErrors(j);
                        if (err !== '') {
                            manageAPIKeys.msgLoad = err;
                            manageAPIKeys.msgLoadType = msgTypes.danger;
                            return;
                        }

                        manageAPIKeys.keys = j.Data || [];
                        manageAPIKeys.keysRetrieved = true;
                        return;
                    })
                    .catch(function (err) {
                        console.log("fetch() error: >>", err, "<<");
                        manageAPIKeys.msgLoad = 'An unknown error occured. Please try again.';
                        manageAPIKeys.msgLoadType = msgTypes.danger;
                        return;
                    });

                return;
            },

            //passToModal handles the clicking of the button/icon that opens the 
            //add/edit API Key modal. If an API key is being edited/viewed, this 
            //passes the chosen API key's data to the modal. If the user wants to
            //get a new API key, this passes "undefined" which will cause the modal
            //to be displayed in an "add" state.
            /**
             * 
             * @param item - An API key's data, or undefined if adding/getting a new API key.
             */
            passToModal: function (item: apiKey | undefined) {
                modalAPIKey.setModalData(item);
                return;
            }
        },
        mounted() {
            //Get list of existing API keys on page load.
            this.getKeys();
            return;
        }
    });
}

if (document.getElementById("modal-apiKey")) {
    //modalAPIKey handles the modal for adding/generating a new API key or for
    //viewing/editing and existing API key.
    //@ts-ignore cannot find name Vue
    var modalAPIKey = new Vue({
        name: 'modalAPIKey',
        delimiters: ['[[', ']]'],
        el: '#modal-apiKey',
        data: {
            //Modal state. Set in setModalData.
            addingNew: false,

            //API key data, either populated for an existing key in setModalData or
            //will be populated when a new key is added/generated.
            key: {} as apiKey,

            //errors
            submitting: false,
            msg: '',
            msgType: '',

            //endpoints
            urls: {
                generate: "/api/api-keys/generate/",
                revoke: "/api/api-keys/revoke/",
            }
        },
        methods: {
            //setModalData populates the modal with data about a chosen API key or
            //sets the modal to an "add" state by reseting it.
            //
            //This function is called by manageAPIKeys.passToModal() when a button
            //is clicked to edit an API key or to add/generate a new one.
            setModalData: function (key: apiKey | undefined) {
                //Always reset to start.
                this.resetModal();

                //User wants to add/generate a new API key, set modal to blank state.
                if (key === undefined) {
                    this.addingNew = true;
                    return;
                }

                //User wants to view/edit an API key.
                this.addingNew = false;
                this.key = key;

                return;
            },

            //resetModal resets the modal back to the blank state used for
            //adding/generating a new API key.
            //
            //This function is called in setModalData() whenever the modal is launched
            //to make sure any previous key's data is removed, and after a key was
            //generated.
            resetModal: function () {
                this.key = {
                    Description: "",
                    K: "",
                } as apiKey;

                this.msg = "";
                this.msgType = "";
                this.submitting = false;
                return;
            },

            //generate adds/generates a new API key.
            generate: function () {
                //Make sure data isn't already being saved.
                if (this.submitting) {
                    console.log("already submitting");
                    return;
                }

                //Validate.
                this.msgType = msgTypes.danger;
                if (this.key.Description === "") {
                    this.msg = "You must provide a description for this API key so you can recognize what it is used for.";
                    return;
                }

                //Validation ok.
                this.msgType = msgTypes.primary;
                this.msg = "Generating...";
                this.submitting = true;

                //Make sure some fields are set to default state.
                this.key.K = '';

                //Perform API call.
                //
                //We are passing in an object here (this.key) instead of just 
                //"description" since we may add stuff in the future.
                let data: Object = {
                    data: JSON.stringify(this.key),
                };
                fetch(post(this.urls.generate, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then(function (j) {
                        //check if response is an error from the server
                        let err: string = handleAPIErrors(j);
                        if (err !== '') {
                            modalAPIKey.msg = err;
                            modalAPIKey.msgType = msgTypes.danger;
                            modalAPIKey.submitting = false;
                            return;
                        }

                        //Hide "generating..." message.
                        modalAPIKey.msg = '';

                        //Show successfully generated new API key in modal so user
                        //can copy it.
                        modalAPIKey.key = j.Data;

                        //Refresh table of API keys.
                        manageAPIKeys.getKeys();

                        //Show just generated message.
                        modalAPIKey.msg = "Your API key was generated successfully. Please copy and use the Key exactly as it is displayed.";
                        modalAPIKey.msgType = msgTypes.primary;

                        //Submitting is never reset to "false" here. User has to 
                        //close and reopen modal to reset form to add/generate another
                        //new API key.
                        //
                        //This was done, and is different from how the rest of the
                        //app functions (wiping form after a short delay), since 
                        //in most cases a user isn't creating a whole bunch of API
                        //keys one after another.
                        return;
                    })
                    .catch(function (err) {
                        console.log("fetch() error: >>", err, "<<");
                        modalAPIKey.msg = 'An unknown error occured. Please try again.';
                        modalAPIKey.msgType = msgTypes.danger;
                        modalAPIKey.submitting = false;
                        return;
                    });
            },

            //revoke marks an API key as inactive. Inactive API keys cannot be used
            //and cannot be reactivated.
            revoke: function (id: number) {
                //Make sure data isn't already being saved.
                if (this.submitting) {
                    console.log("already submitting");
                    return;
                }

                //Validate.
                if (id < 1) {
                    this.msg = 'Could not determine which API key to revoke';
                    this.msgType = msgTypes.danger;
                    return;
                }

                //Validation ok.
                this.msgType = msgTypes.primary;
                this.msg = "Revoking...";
                this.submitting = true;

                //Perform API call.
                let data: Object = {
                    id: this.key.ID,
                };
                fetch(post(this.urls.revoke, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then(function (j) {
                        //check if response is an error from the server
                        let err: string = handleAPIErrors(j);
                        if (err !== '') {
                            modalAPIKey.msg = err;
                            modalAPIKey.msgType = msgTypes.danger;
                            modalAPIKey.submitting = false;
                            return;
                        }

                        //Refresh table of API keys.
                        manageAPIKeys.getKeys();

                        //Clear out the API key from being displayed in the modal,
                        //even though modal is going to be closed.
                        modalAPIKey.key.K = "";

                        //Close the modal.
                        //@ts-ignore modal does not exist on jQuery.
                        $('#' + modalAPIKey.$el.id).modal('hide');

                        return;
                    })
                    .catch(function (err) {
                        console.log("fetch() error: >>", err, "<<");
                        modalAPIKey.msgSave = 'An unknown error occured. Please try again.';
                        modalAPIKey.msgSaveType = msgTypes.danger;
                        modalAPIKey.submitting = false;
                        return;
                    });
            },
        }
    });
}
