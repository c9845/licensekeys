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
            //Handle GUI states:
            // - Page just loaded and no API key selected: show lookup select inputs 
            //   (uses addingNew & keyData.ID).
            // - User wants to add/create API key: show add/edit inputs.
            // - User chose an existing API key: show lookup select inputs & add/edit 
            //   inputs. (uses addingNew & keyData.ID)
            addingNew: false,

            //List of API keys.
            keys: [] as apiKey[],
            keysRetrieved: false,

            //Errors when loading API keys.
            msgLoad: '',
            msgLoadType: '',

            //Single API key selected.
            apiKeySelectedID: 0,
            keyData: {
                Description: "",
                K: "",
            } as apiKey,

            //Handle confirmation of revoke, so a single click cannot revoke an API
            //key.
            showRevokeConfirm: false,

            //Errors when saving.
            submitting: false,
            msgSave: '',
            msgSaveType: '',

            //Endpoints.
            urls: {
                getKeys: "/api/api-keys/",
                generate: "/api/api-keys/generate/",
                revoke: "/api/api-keys/revoke/",
                update: "/api/api-keys/update/",
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

                return "View API Key";
            },
        },
        methods: {
            //getKeys looks up the list of existing, active, API keys.
            getKeys: function () {
                let data: Object = {};
                fetch(get(this.urls.getKeys, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then(function (j) {
                        //Check if response is an error from the server.
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
                        manageAPIKeys.msgLoad = 'An unknown error occurred. Please try again.';
                        manageAPIKeys.msgLoadType = msgTypes.danger;
                        return;
                    });

                return;
            },

            //setField saves a chosen radio toggle to the Vue object.
            setField: function (fieldName: string, value: boolean) {
                //Save the field being clicked on value.
                this.keyData[fieldName] = value;
                return;
            },

            //setUIState handles setting the GUI state to the "add" or "edit/view" 
            //states per user clicks by setting some variables. This also resets any 
            //inputs to empty/default values as needed. Basically, we flip the value 
            //of the addingNew variable and clear some inputs.
            setUIState: function () {
                //User clicked on "lookup" button, user wants to see "lookup" UI. We
                //don't have to worry about setting the inputs to since showAPIKey()
                //will be called on choosing a select option and populate the inputs
                //accordingly.
                if (this.addingNew) {
                    this.addingNew = !this.addingNew;
                    return;
                }

                //User clicked on "add" button, user wants to see "add" form. Clear 
                //any existing user data so that the inputs are reset to a blank state.
                this.addingNew = !this.addingNew;
                this.resetKeyData();

                this.msgSave = '';
                this.msgSaveType = '';

                return;
            },

            //resetKeyData resets the inputs and toggles when showing the "add user"
            //GUI, or after a user is added and we empty all the inputs.
            resetKeyData: function () {
                this.keyData = {
                    Description: "",
                    K: "",
                } as apiKey;
                this.apiKeySelectedID = 0;
                this.showRevokeConfirm = false;

                //Set all the toggles to default states.
                //@ts-ignore Vue doesn't exist
                Vue.nextTick(function () {
                    for (let key in (manageAPIKeys.keyData as apiKey)) {
                        let value = manageAPIKeys.keyData[key];
                        if (value === true || value === false) {
                            setToggle(key, value);
                        }
                    }
                });

                return;
            },

            //showAPIKey populates the "lookup" GUI with data about the chosen API
            //key. This is called when an API key is chosen from the list of retrieved 
            //API keys. This function is needed in order to set any toggles properly 
            //since we cannot just do that with Vue v-model.
            showAPIKey: function () {
                //Make sure an API key was selected. This handles the "Please choose"
                //option or something similar.
                if (this.apiKeySelectedID < 1) {
                    return;
                }

                //Get data from the list of API keys.
                for (let k of (this.keys as apiKey[])) {
                    if (k.ID !== this.apiKeySelectedID) {
                        continue;
                    }

                    //Save the chosen user for displaying in the GUI.
                    this.keyData = k;

                    //Set toggles.
                    //@ts-ignore cannot find Vue
                    Vue.nextTick(function () {
                        for (let key in k) {
                            let value = k[key];
                            if (value === true || value === false) {
                                setToggle(key, value);
                            }
                        }
                    });
                }

                this.showRevokeConfirm = false;

                return;
            },

            //createKey adds/generates a new API key.
            createKey: function () {
                //Make sure data isn't already being saved.
                if (this.submitting) {
                    console.log("already submitting");
                    return;
                }

                //Validate.
                this.msgSaveType = msgTypes.danger;
                if (this.keyData.Description === "") {
                    this.msgSave = "You must provide a description for this API key so you can recognize what it is used for.";
                    return;
                }

                //Validation ok.
                this.msgSaveType = msgTypes.primary;
                this.msgSave = "Creating API Key...";
                this.submitting = true;

                //Make sure some fields are set to default state.
                this.keyData.K = '';

                //Make API request.
                //
                //We are passing in an object here (this.key) instead of just 
                //"description" since we may add stuff in the future.
                let data: Object = {
                    data: JSON.stringify(this.keyData),
                };
                fetch(post(this.urls.generate, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then(function (j) {
                        //Check if response is an error from the server.
                        let err: string = handleAPIErrors(j);
                        if (err !== '') {
                            manageAPIKeys.msgSave = err;
                            manageAPIKeys.msgSaveType = msgTypes.danger;
                            manageAPIKeys.submitting = false;
                            return;
                        }


                        //Refresh the list of keys in the GUI.
                        manageAPIKeys.getKeys();

                        //Show the just created key.
                        setTimeout(function () {
                            //Show success message.
                            manageAPIKeys.msgSaveType = "";
                            manageAPIKeys.msgSave = "";
                            manageAPIKeys.submitting = false;

                            //Select this just created key.
                            manageAPIKeys.apiKeySelectedID = j.Data;

                            //Show the key for copying.
                            manageAPIKeys.addingNew = false;
                            manageAPIKeys.showAPIKey();
                        }, defaultTimeout);

                        return;
                    })
                    .catch(function (err) {
                        console.log("fetch() error: >>", err, "<<");
                        manageAPIKeys.msgSave = 'An unknown error occurred. Please try again.';
                        manageAPIKeys.msgSaveType = msgTypes.danger;
                        manageAPIKeys.submitting = false;
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
                    this.msgSave = 'Could not determine which API key to revoke';
                    this.msgSaveType = msgTypes.danger;
                    return;
                }

                //Validation ok.
                this.msgSaveType = msgTypes.danger;
                this.msgSave = "Revoking...";
                this.submitting = true;

                //Make API request.
                let data: Object = {
                    id: this.apiKeySelectedID,
                };
                fetch(post(this.urls.revoke, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then(function (j) {
                        //Check if response is an error from the server.
                        let err: string = handleAPIErrors(j);
                        if (err !== '') {
                            manageAPIKeys.msgSave = err;
                            manageAPIKeys.msgSaveType = msgTypes.danger;
                            manageAPIKeys.submitting = false;
                            return;
                        }

                        //Show revoke message.
                        manageAPIKeys.msgSaveType = msgTypes.primary
                        manageAPIKeys.msgSave = "API key was revoked successfully!"


                        setTimeout(function () {
                            manageAPIKeys.msgSaveType = "";
                            manageAPIKeys.msgSave = "";
                            manageAPIKeys.submitting = false;

                            manageAPIKeys.apiKeySelectedID = 0;
                            manageAPIKeys.resetKeyData();

                            manageAPIKeys.getKeys();
                        }, defaultTimeout);

                        return;
                    })
                    .catch(function (err) {
                        console.log("fetch() error: >>", err, "<<");
                        manageAPIKeys.msgSave = 'An unknown error occurred. Please try again.';
                        manageAPIKeys.msgSaveType = msgTypes.danger;
                        manageAPIKeys.submitting = false;
                        return;
                    });
            },

            //handleRevokeConfirm handles flipping the field that then shows/hides
            //the "confirm" button when revoking an API key. This "confirm" button
            //was implemented to prevent accidental deletion/revokes of an API key.
            //This forces a user to click two buttons to revoke an API key which is
            //must less likely to be accidental.
            //
            //The "confirm" button is reverted after a short amount of time so that
            //a user must perform the "confirm" quickly.
            handleRevokeConfirm: function () {
                this.showRevokeConfirm = true;

                setTimeout(function () {
                    manageAPIKeys.showRevokeConfirm = false;
                }, 3000);

                return;
            },

            //update saves changes to an API key's description or permissions. You
            //cannot change the actual API key.
            update: function () {
                //Make sure data isn't already being saved.
                if (this.submitting) {
                    console.log("already submitting");
                    return;
                }

                //Validate.
                this.msgSaveType = msgTypes.danger;
                if (this.keyData.ID < 1) {
                    this.msgSave = "Could not determine which API Key you want to update.";
                    return;
                }

                //Validation ok.
                this.msgSaveType = msgTypes.primary;
                this.msgSave = "Saving...";
                this.submitting = true;

                //Make API request.
                let data: Object = {
                    data: JSON.stringify(this.keyData),
                };
                fetch(post(this.urls.update, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then(function (j) {
                        //Check if response is an error from the server.
                        let err: string = handleAPIErrors(j);
                        if (err !== '') {
                            manageAPIKeys.msgSave = err;
                            manageAPIKeys.msgSaveType = msgTypes.danger;
                            manageAPIKeys.submitting = false;
                            return;
                        }

                        //Show success.
                        manageAPIKeys.msgSaveType = msgTypes.primary;
                        manageAPIKeys.msgSave = "Changes saved!";
                        setTimeout(function () {
                            //Show success message.
                            manageAPIKeys.msgSaveType = "";
                            manageAPIKeys.msgSave = "";
                            manageAPIKeys.submitting = false;
                        }, defaultTimeout);

                        return;
                    })
                    .catch(function (err) {
                        console.log("fetch() error: >>", err, "<<");
                        manageAPIKeys.msgSave = 'An unknown error occurred. Please try again.';
                        manageAPIKeys.msgSaveType = msgTypes.danger;
                        manageAPIKeys.submitting = false;
                        return;
                    });
            }
        },
        mounted() {
            //Get list of existing API keys on page load.
            this.getKeys();
            return;
        }
    });
}
