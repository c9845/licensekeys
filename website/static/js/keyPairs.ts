/**
 * keyPairs.ts
 * This file handles adding, viewing, and deleting the key pairs assinged to an app for 
 * signing the license file data.
 */


/// <reference path="common.ts" />
/// <reference path="fetch.ts" />
/// <reference path="types.ts" />

if (document.getElementById("listKeyPairs")) {
    //listKeyPairs handles displaying the list of key pairs for an app. This does not
    //handle adding a new key pair which is done in a modal.
    //@ts-ignore cannot find name Vue
    var listKeyPairs = new Vue({
        name: 'listKeyPairs',
        delimiters: ['[[', ']]'],
        el: '#listKeyPairs',
        data: {
            //App to look up key pairs for. This is populated by setAppID.
            appSelectedID: 0,

            //List of key pairs for this app.
            keyPairs:           [] as keyPair[],
            keyPairsRetrieved:  false,

            //errors
            msgLoad:        '',
            msgLoadType:    '',

            collapseUI: false, //collapse the card to take up less screen space.

            //endpoints
            urls: {
                get: "/api/key-pairs/",
            }
        },
        methods: {
            //setAppID sets the appSelectedID value in this vue object. This is called from
            //manageApps.setAppInOtherVueObjects() when an app is chosen from the list of
            //defined apps. This then retrieves the list of key pairs for this app.
            setAppID: function(appID: number) {
                this.appSelectedID = appID;
                
                //handle adding a new app.
                if (appID === 0) {
                    this.keyPairs = [];
                    this.msgLoad =  "";
                    return;
                }
                
                this.getKeyPairs();
                return;
            },

            //getKeyPairs gets the list of key pairs that have been defined for this app.
            getKeyPairs: function () {
                let data: Object = {
                    appID:      this.appSelectedID,
                    activeOnly: true,
                };
                fetch(get(this.urls.get, data))
                .then(handleRequestErrors)
                .then(getJSON)
                .then(function (j) {
                    //check if response is an error from the server
                    let err: string = handleAPIErrors(j);
                    if (err !== '') {
                        listKeyPairs.msgLoad =     err;
                        listKeyPairs.msgLoadType = msgTypes.danger;
                        return;
                    }
    
                    //save data to display in gui
                    listKeyPairs.keyPairs =             j.Data || [];
                    listKeyPairs.keyPairsRetrieved =    true;
    
                    return;
                })
                .catch(function (err) {
                    console.log("fetch() error: >>", err, "<<");
                    listKeyPairs.msgLoad =     'An unknown error occured. Please try again.';
                    listKeyPairs.msgLoadType = msgTypes.danger;
                    return;
                });
    
                return;
            },

            //passToModal handles the clicking of buttons/icons that open the add/view key pair
            //modal. When adding a new key pair, 'undefined' is simply passed along. But when
            //viewing the full details of a key pair (full data was already retrieved to show
            //list of key pairs), this passes along an object representing the key pair's data.
            //This is done so that we don't need to retrieve the full data about a key pair again
            //and we can reuse the same modal for adding or viewing.
            //
            //Note that this does not open the modal. Opening of the modal is handled through
            //bootstrap data-toggle and data-target attributes.
            passToModal: function(item: keyPair | undefined) {
                modalKeyPair.setModalData(item);
                return
            },
        }
    })
}

if (document.getElementById("modal-keyPair")) {
    //modalKeyPair handles displaying the full details of a keypair, adding a new keypair,
    //deleting a keypair, and downloading the public key from the keypair.
    //@ts-ignore cannot find name Vue
    var modalKeyPair = new Vue({
        name: 'modalKeyPair',
        delimiters: ['[[', ']]'],
        el: '#modal-keyPair',
        data: {
            //App the keypair is for. This is populated by setAppID. This is mostly used for
            //setting the modal form to a default state for adding a new keypair.
            appSelectedID: 0,

            //keypair data. Populated by setModalData with either blank data when adding a
            //new keypair or full data about an existing keypair when viewing/deleting/getting
            //public key.
            keyPairData: {} as keyPair,

            //options to choose from when adding and default
            algorithmTypes:         keyPairAlgoTypes,
            defaultAlgorithmType:   keyPairAlgoED25519,

            showPublicKey: false, //true upon button click to show public key in textarea for copying
            
            //errors
            submitting:     false,
            msgSave:        '',
            msgSaveType:    '',

            //endpoints
            urls: {
                add:            "/api/key-pairs/add/",
                delete:         "/api/key-pairs/delete/",
                setDefault:     "/api/key-pairs/set-default/",
            }
        },
        computed: {
            //adding is set to true when the user is adding/generating a key pair. This is set
            //using the keypair's ID which is only set when looking up an existing key pair. This
            //is used to modify what the GUI displays (modal title and body text) to show correct
            //helpful information based on adding or viewing a keypair.
            adding: function () {
                if (this.keyPairData.ID === undefined || this.keyPairData.ID < 1) {
                    return true;
                }
    
                return false;
            },

            //publicKeyNumLines returns the number of lines in the public key and is
            //used to set the "rows" attribute on the textarea so that the entire
            //public key is visible for ease of copying.
            publicKeyNumLines: function() {
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
            //setAppID sets the appSelectedID value in this vue object. This is called from
            //manageApps.setAppInOtherVueObjects() when an app is chosen from the list of
            //defined apps.
            setAppID: function(appID: number) {
                this.appSelectedID = appID;
                return;
            },

            //setModalData is used to populate the modal with data from the clicked keypair
            //in the list of keypairs. This is also used to reset the modal to a clean state
            //when adding a new keypair.
            setModalData: function(item: keyPair | undefined) {
                //Always reset.
                this.resetModal();
                
                //User wants to add a new keypair.
                if (item === undefined) {
                    return;
                }
                
                //user is viewing details of a keypair.
                this.keyPairData =  item
                return;
            },

            //resetModal sets the modal back to a clean state for adding a new keypair.
            resetModal: function() {
                this.keyPairData = {
                    ID:             0,
                    DatetimeCreated:    "", //wont be set, just to match type.
                    DatetimeModified:   "", //" " "
                    CreatedByUserID:    0,  //" " "
                    Active:             true,
                    AppID:              this.appSelectedID,
                    Name:               "",
                    AlgorithmType:      this.defaultAlgorithmType,
                    PublicKey:          "",
                    IsDefault:          false,
                } as keyPair;

                this.showPublicKey = false;

                this.submitting =   false;
                this.msgSave =      "";
                this.msgSaveType =  "";
                return;
            },

            //add saves a new keypair. The details in the form are submitting and the server
            //will create the new keypair and save it to this app's database. You can then 
            //download the public key. The list of keypairs will also be updated (in parent
            //card).
            add: function() {
                //make sure data isn't already being submitted
                if (this.submitting) {
                    console.log("already submitting...");                
                    return;
                }

                //validate
                this.msgSaveType = msgTypes.danger;
                if (this.keyPairData.Name.trim() === "") {
                    this.msgSave = "You must provide a name for this key pair.";
                    return;
                }
                if (!this.algorithmTypes.includes(this.keyPairData.AlgorithmType)) {
                    this.msgSave = "Please choose an algorithm from the provided options";
                    return;
                }

                //validation ok
                this.msgSave =      "Generating key pair...";
                this.msgSaveType =  msgTypes.primary;
                this.submitting =   true;

                //perform api call
                let data: Object = {
                    data: JSON.stringify(this.keyPairData),
                };
                fetch(post(this.urls.add, data))
                .then(handleRequestErrors)
                .then(getJSON)
                .then(function (j) {
                    //check if response is an error from the server
                    let err: string = handleAPIErrors(j);
                    if (err !== '') {
                        modalKeyPair.msgSave =        err;
                        modalKeyPair.msgSaveType =    msgTypes.danger;
                        modalKeyPair.submitting =     false;
                        return;
                    }

                    //Update the local key pair data with data from the server. This
                    //includes the keypair's ID and public key. We do this, instead
                    //of just getting the keypair's ID back so that we can display
                    //the public key for copying without having to make another API
                    //call to retrieve the data.
                    modalKeyPair.keyPairData = j.Data;

                    //Refresh the list of keypairs so that this new keypair is shown.
                    listKeyPairs.getKeyPairs();

                    //Show success message briefly.
                    modalKeyPair.msgSave =        "Added!";
                    modalKeyPair.msgSaveType =    msgTypes.success;
                    modalKeyPair.submitting =     false;
                    setTimeout(function () {
                        modalKeyPair.msgSave =        '';
                        modalKeyPair.msgSaveType =    '';
                    }, defaultTimeout);

                    return;
                })
                .catch(function (err) {
                    console.log("fetch() error: >>", err, "<<");
                    modalKeyPair.msgSave =        'An unknown error occured.  Please try again.';
                    modalKeyPair.msgSaveType =    msgTypes.danger;
                    modalKeyPair.submitting =     false;
                    return;
                });

                return;
            },

            //remove marks a keypair as inactive. Inactive keypair will no long show up in
            //the list of keypairs or be available for use when creating a new license.
            remove: function() {
                //make sure data isn't already being submitted
                if (this.submitting) {
                    console.log("already submitting...");                
                    return;
                }
                
                //Make sure we know what field we are deleting.
                if (isNaN(this.keyPairData.ID) || this.keyPairData.ID === '' || this.keyPairData.ID < 1) {
                    this.msgSave =      "Could not determine which key pair you are trying to delete. Please refresh the page and try again.";
                    this.msgSaveType =  msgTypes.danger;
                    return;
                }
    
                //validation ok
                this.msgSave =      "Deleting...";
                this.msgSaveType =  msgTypes.primary;
                this.submitting =   true;
    
                //perform api call
                let data: Object = {
                    id: this.keyPairData.ID,
                };
                fetch(post(this.urls.delete, data))
                .then(handleRequestErrors)
                .then(getJSON)
                .then(function (j) {
                    //check if response is an error from the server
                    let err: string = handleAPIErrors(j);
                    if (err !== '') {
                        modalKeyPair.msgSave =        err;
                        modalKeyPair.msgSaveType =    msgTypes.danger;
                        return;
                    }
                    
                    //refresh the list of keypairs in table. The modal will
                    //be closed by the data-dismiss on the button clicked.
                    listKeyPairs.getKeyPairs();
                    modalKeyPair.submitting = false;

                    return;
                })
                .catch(function (err) {
                    console.log("fetch() error: >>", err, "<<");
                    modalKeyPair.msgSave =        'An unknown error occured. Please try again.';
                    modalKeyPair.msgSaveType =    msgTypes.danger;
                    modalKeyPair.submitting =     false;
                    return;
                });
                return;
            },

            //setDefault marks this keypair as the default for the app. This can only be
            //done for non-default keypairs.
            setDefault: function() {
                //make sure data isn't already being submitted
                if (this.submitting) {
                    console.log("already submitting...");                
                    return;
                }
                
                //Make sure we know what field we are deleting.
                if (isNaN(this.keyPairData.ID) || this.keyPairData.ID === '' || this.keyPairData.ID < 1) {
                    this.msgSave =      "Could not determine which key pair you want to set as default. Please refresh the page and try again.";
                    this.msgSaveType =  msgTypes.danger;
                    return;
                }
    
                //validation ok
                this.msgSave =      "Setting default...";
                this.msgSaveType =  msgTypes.primary;
                this.submitting =   true;
    
                //perform api call
                let data: Object = {
                    id: this.keyPairData.ID,
                };
                fetch(post(this.urls.setDefault, data))
                .then(handleRequestErrors)
                .then(getJSON)
                .then(function (j) {
                    //check if response is an error from the server
                    let err: string = handleAPIErrors(j);
                    if (err !== '') {
                        modalKeyPair.msgSave =        err;
                        modalKeyPair.msgSaveType =    msgTypes.danger;
                        return;
                    }
                    
                    //set the default field for this keypair to update gui.
                    //also refresh the list of keypairs to show the default
                    //tag next to the correct keypair
                    listKeyPairs.getKeyPairs();
                    modalKeyPair.keyPairData.IsDefault = true;
                    modalKeyPair.submitting = false;
                    modalKeyPair.msgSave = "";

                    return;
                })
                .catch(function (err) {
                    console.log("fetch() error: >>", err, "<<");
                    modalKeyPair.msgSave =        'An unknown error occured. Please try again.';
                    modalKeyPair.msgSaveType =    msgTypes.danger;
                    modalKeyPair.submitting =     false;
                    return;
                });
                return;
            }
        },
        mounted() {
            //this is used to set the object storing keypair data to a default state
            this.resetModal();

            return;
        }
    })
}