/**
 * apikeys.ts
 * The code in this is used on the manage api keys page.
 * This is used for getting a list of api keys, adding new
 * keys, or revoking old keys.
 */

/// <reference path="common.ts" />
/// <reference path="fetch.ts" />
/// <reference path="types.ts" />

if (document.getElementById("manageAPIKeys")) {
    //manageAPIKeys handles the loading of data for the api keys page.  It
    //also handles revoking/deleting an api key.
    //@ts-ignore cannot find name Vue
    var manageAPIKeys = new Vue({
        name: 'manageAPIKeys',
        delimiters: ['[[', ']]'],
        el: '#manageAPIKeys',
        data: {
            //list of api keys
            keys:           [] as apiKey[],
            keysRetrieved:  false,
            
            //errors
            submitting:     false,
            msg:            '',
            msgType:        '',

            //endpoints
            urls: {
                get:    "/api/api-keys/",
            }
        },
        methods: {
            //getKeys gets the list of api keys
            getKeys: function () {
                let data: Object = {};
                fetch(get(this.urls.get, data))
                .then(handleRequestErrors)
                .then(getJSON)
                .then(function (j) {
                    //check if response is an error from the server
                    let err: string = handleAPIErrors(j);
                    if (err !== '') {
                        manageAPIKeys.msgLoad =     err;
                        manageAPIKeys.msgLoadType = msgTypes.danger;
                        return;
                    }
    
                    manageAPIKeys.keys =            j.Data || [];
                    manageAPIKeys.keysRetrieved =   true;
                    return;
                })
                .catch(function (err) {
                    console.log("fetch() error: >>", err, "<<");
                    manageAPIKeys.msgLoad =     'An unknown error occured.  Please try again.';
                    manageAPIKeys.msgLoadType = msgTypes.danger;
                    return;
                });
    
                return;
            },
    
            //passToModal handles the clicking of buttons/icons that open the add/view apikey
            //field modal. When adding a new api key, 'undefined' is simply passed along. 
            //But when viewing the details of an api key (full data was already retrieved 
            //to show list of api keys), this passes along an object representing the api key's
            //data. This is done so that we don't need to retrieve the full data again and we 
            //can reuse the same modal for adding or viewing.
            //
            //Note that this does not open the modal. Opening of the modal is handled through
            //bootstrap data-toggle and data-target attributes.
            passToModal: function(item: keyPair | undefined) {
                modalAPIKey.setModalData(item);
                return
            },
        },
        mounted() {
            if (this.$el.id !== "" && document.getElementById(this.$el.id)) {
                this.getKeys();
            }
    
            return;
        }
    });
}

//modalAPIKey handles generating a new api key or viewing details of an api key.
if (document.getElementById("modal-apiKey")) {
    //@ts-ignore cannot find name Vue
    var modalAPIKey = new Vue({
        name: 'modalAPIKey',
        delimiters: ['[[', ']]'],
        el: '#modal-apiKey',
        data: {
            //api key data
            keyData:                    {} as apiKey,
            keyGeneratedSuccessfully:   false,
    
            //errors
            submitting: false,
            msg:        '',
            msgType:    '',

            //endpoints
            urls: {
                generate:   "/api/api-keys/generate/",
                revoke:     "/api/api-keys/revoke/",
            }
        },
        computed: {
            //adding is set to true when the user is adding an api key. This is set
            //using the key's ID which is only set when looking up an existing key. This
            //is used to modify what the GUI displays (modal title and body text) to show 
            //correct helpful information based on adding or editing a field.
            adding: function () {
                if (this.keyData.ID === undefined || this.keyData.ID < 1) {
                    return true;
                }
    
                return false;
            },
        },
        methods: {
            //setModalData is used to populate the modal with data from the clicked api
            //key in the list of keys. This is also used to reset the modal to a clean 
            //state when adding a new api key.
            setModalData: function(item: apiKey | undefined) {
                this.resetModal();

                //user wants to add a new api key.
                if (item === undefined) {
                    return
                }

                //user is viewing details of an api key.
                this.keyData =  item
                return;
            },

            //resetModal resets the modal back to the original "add" state
            //This is called when the modal is opened by using the "callResetModal" func on defined
            //for the Vue instance that manages the table.
            resetModal: function() {
                this.keyData =                  {} as apiKey;
                this.keyGeneratedSuccessfully = false;
                this.submitting =               false;
                this.msg =                      "";
                return;
            },
    
            //generate creates a new API key.
            generate: function() {
                if (this.submitting) {
                    console.log("already submitting");
                    return;
                }
                
                //validate
                this.msgType = msgTypes.danger;
                if (this.keyData.Description === "") {
                    this.msg = "You must provide a description for this API key so you can recognize what it is used for.";
                    return;
                }
    
                //make sure data isn't already being saved
                if (this.submitting) {
                    console.log("already submitting...");                
                    return;
                }
    
                //validation ok
                this.msgType =                  msgTypes.primary;
                this.msg =                      "Generating...";
                this.keyGeneratedSuccessfully = false;
                this.keyData.K =                ''; //empty out even though this isn't needed.
                this.submitting =               true;
    
                //Perform API request.
                let data: Object = {
                    data: JSON.stringify(this.keyData),
                };
                fetch(post(this.urls.generate, data))
                .then(handleRequestErrors)
                .then(getJSON)
                .then(function (j) {
                    //check if response is an error from the server
                    let err: string = handleAPIErrors(j);
                    if (err !== '') {
                        modalAPIKey.msg =        err;
                        modalAPIKey.msgType =    msgTypes.danger;
                        modalAPIKey.submitting = false;
                        return;
                    }
    
                    //hide "generating..." message
                    modalAPIKey.msg = '';
    
                    //Show api key in modal.
                    modalAPIKey.key =                        j.Data;
                    modalAPIKey.keyGeneratedSuccessfully =   true;
    
                    //refresh table of api keys
                    manageAPIKeys.getKeys();
    
                    //start timer to wipe the key from gui
                    //Note the long timeout duration.
                    setTimeout(function() {
                        modalAPIKey.resetModal();
                    }, 60 * 1000);
    
                    modalAPIKey.submitting = false;
                    return;
                })
                .catch(function (err) {
                    console.log("fetch() error: >>", err, "<<");
                    modalAPIKey.msg =        'An unknown error occured.  Please try again.';
                    modalAPIKey.msgType =    msgTypes.danger;
                    modalAPIKey.submitting = false;
                    return;
                });
            },

            //revoke handles revoking an API key. This marks an API key as inactive.
            //The ID provided is the key's database ID, not the KeyID.
            revoke: function() {
                if (this.submitting) {
                    console.log("already submitting");
                    return;
                }
                
                //Validation.
                if (this.keyData.ID < 1) {
                    this.msg =      'Could not determine which API key to revoke';
                    this.msgType =  msgTypes.danger;
                    return;
                }

                //validation ok
                this.msg =          "Revoking...";
                this.msgType =      msgTypes.primary;
                this.submitting =   true;
    
                //Perform API request.
                let data: Object = {
                    id: this.keyData.ID,
                };
                fetch(post(this.urls.revoke, data))
                .then(handleRequestErrors)
                .then(getJSON)
                .then(function (j) {
                    //check if response is an error from the server
                    let err: string = handleAPIErrors(j);
                    if (err !== '') {
                        manageAPIKeys.msg =         err;
                        manageAPIKeys.msgType =     msgTypes.danger;
                        manageAPIKeys.submitting =  false;
                        return;
                    }
    
                    //refresh the list of api keys in table. The modal will
                    //be closed by the data-dismiss on the button clicked.
                    manageAPIKeys.getKeys();
                    manageAPIKeys.submitting = false;
                    return;
                })
                .catch(function (err) {
                    console.log("fetch() error: >>", err, "<<");
                    manageAPIKeys.msgSave =     'An unknown error occured.  Please try again.';
                    manageAPIKeys.msgSaveType = msgTypes.danger;
                    manageAPIKeys.submitting =  false;
                    return;
                });
            }
        }
    });
}
