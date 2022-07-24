/**
 * users.ts
 * The code in this is used on the manage users page.
 */

/// <reference path="common.ts" />
/// <reference path="fetch.ts" />
/// <reference path="types.ts" />

if (document.getElementById("manageUsers")) {
    //users is used to add or modify a user
    //@ts-ignore cannot find name Vue
    var manageUsers = new Vue({
        name: 'manageUsers',
        delimiters: ['[[', ']]'],
        el: '#manageUsers',
        data: {
            //handle gui states:
            // - page just loaded and no user selected: show lookup select inputs (uses addingNew & userData.ID)
            // - user wants to add user: show add/edit inputs.
            // - user chose a user: show lookup select inputs & add/edit inputs. (uses addingNew & userData.ID)
            addingNew: false,
    
            //errors when loading users
            msgLoad:     '',
            msgLoadType: '',
    
            //list of users
            users:          [] as user[],
            usersRetrieved: false,
    
            //single user selected
            userSelectedID: 0,
            userData:       {} as user,
    
            //adding or updating
            submitting:     false,
            msgSave:        '',
            msgSaveType:    '',
    
            minPasswordLength: 10, //set in mounted by reading hidden input

            //need multiple password inputs for when adding new user
            password1: '',
            password2: '',

            //forcing logout
            forceLogoutBtnText:     "Force Logout",
            forceLogoutSubmitting:  false,

            //endpoints
            urls: {
                get:         "/api/users/",
                add:         "/api/users/add/",
                update:      "/api/users/update/",
                forceLogout: "/api/users/force-logout/",
            }
        },
        computed: {
            //addEditCardTitle sets the text of the card used for adding or editing.
            //Since the card used for adding & editing is the same we want to show the 
            //correct card title to the user so they know what they are doing.
            addEditCardTitle: function() {
                if (this.addingNew) {
                    return "Add User";
                }
    
                return "Edit User";
            },
        },
        methods: {
            //getUsers gets the list of users
            getUsers: function () {
                let data: Object = {};
                fetch(get(this.urls.get, data))
                .then(handleRequestErrors)
                .then(getJSON)
                .then(function (j) {
                    //check if response is an error from the server
                    let err: string = handleAPIErrors(j);
                    if (err !== '') {
                        manageUsers.msgLoad =       err;
                        manageUsers.msgLoadType =   msgTypes.danger;
                        return;
                    }
    
                    manageUsers.users =             j.Data || [];
                    manageUsers.usersRetrieved =    true;
                    return;
                })
                .catch(function (err) {
                    console.log("fetch() error: >>", err, "<<");
                    manageUsers.msgLoad =       'An unknown error occured.  Please try again.';
                    manageUsers.msgLoadType =   msgTypes.danger;
                    return;
                });
    
                return;
            },
    
            //setField saves a chosen radio toggle to the Vue object
            setField: function (fieldName: string, value: boolean) {
                this.userData[fieldName] = value;
                return;
            },
    
            //setUIState handles setting the gui state to the "add" or "edit/view" 
            //states per user clicks by setting some variables. This also resets any 
            //inputs to empty/default values as needed. Basically, we flip the value 
            //of the addingNew variable and clear some inputs.
            setUIState: function() {
                //User clicked on "lookup" button, user wants to see "lookup" UI. We
                //don't have to worry about setting the inputs to since showUser()
                //will be called on choosing a select option and populate the inputs
                //accordingly.
                if (this.addingNew) {
                    this.addingNew = !this.addingNew;
                    return;
                }
                
                //User clicked on "add" button, user wants to see "add" form. Clear 
                //any existing user data so that the inputs are reset to a blank state.
                this.addingNew = !this.addingNew;
                this.resetUserData();
    
                this.msgSave =      '';
                this.msgSaveType =  '';
    
                return;
            },

            //resetUserData resets the inputs and toggles when showing the "add user"
            //gui, or after a user is added and we empty all the inputs.
            resetUserData: function() {
                this.userData =         {
                    Username:       "",

                    Active:         false,
                    Administrator:  false,
                    CreateLicenses: false,
                    ViewLicenses:   false,

                    PasswordInput1: "",
                    PasswordInput2: "",
                };
                this.userSelectedID =   0;

                //Set all the toggles to "off" since we set all the fields for the 
                //user data to "false".
                //@ts-ignore Vue doesn't exist
                Vue.nextTick(function () {
                    setToggle('btn-group-toggle', false, true);
                });

                return;
            },

            //showUser populates the "lookup" gui with data about the chosen item. 
            //This is called when a user is chosen from the list of retrieved users. 
            //This function is needed in order to set any toggles properly since we 
            //cannot just do that with Vue v-model.
            showUser: function() {
                //Make sure an item was selected. This handles the "Please choose"
                //option or something similar.
                if (this.userSelectedID < 1) {
                    return;
                }
    
                //Get data from the list of items.
                for (let i of (this.items as user[])) {
                    if (i.ID !== this.userSelectedID)  {
                        continue;
                    }
    
                    //Save the chosen item for displaying in the GUI.
                    this.itemData = i;
    
                    //Set toggles.
                    //@ts-ignore cannot find Vue
                    Vue.nextTick(function() {
                        for (let key in i) {
                            let value = i[key];
                            if (value === true || value === false) {
                                setToggle(key, value);
                            }
                        }
                    });
    
                    //Set the item ID in other Vue objects.
                    modalChangePassword.setUserData(this.userSelectedID);
                    modalActivate2FA.setUserData(this.userSelectedID);
                    modalDeactivate2FA.setUserData(this.userSelectedID);
                }
    
                return;
            },
    
            //update saves changes made to a user.
            update: function () {
                //Validation.
                if (isNaN(this.userData.ID) || this.userData.ID === '' || this.userData.ID < 1) {
                    this.msgSave = "An error occured. Please reload this page and try again.";
                    this.msgSaveType = msgTypes.danger;
                    return;
                }
    
                //Validation ok.
                this.msgSave =      "Saving...";
                this.msgSaveType =  msgTypes.primary;
                this.submitting =   true;
    
                //perform api call
                let data: Object = {
                    data: JSON.stringify(this.userData),
                };
                fetch(post(this.urls.update, data))
                .then(handleRequestErrors)
                .then(getJSON)
                .then(function (j) {
                    //check if response is an error from the server
                    let err: string = handleAPIErrors(j);
                    if (err !== '') {
                        manageUsers.msgSave =       err;
                        manageUsers.msgSaveType =   msgTypes.danger;
                        manageUsers.submitting =    false;
                        return;
                    }
    
                    //Show success message.
                    manageUsers.msgSave =       "Changes saved!";
                    manageUsers.msgSaveType =   msgTypes.success;
                    setTimeout(function () {
                        manageUsers.msgSave =       '';
                        manageUsers.msgSaveType =   '';
                        manageUsers.submitting =    false;
                    }, defaultTimeout);
    
                    return;
                })
                .catch(function (err) {
                    console.log("fetch() error: >>", err, "<<");
                    manageUsers.msgSave =       'An unknown error occured.  Please try again.';
                    manageUsers.msgSaveType =   msgTypes.danger;
                    manageUsers.submitting =    false;
                    return;
                });
    
                return;
            },
    
            //add saves a new user.
            add: function () {
                //Make sure data isn't already being submitted.
                if (this.submitting) {
                    console.log("already submitting...");                
                    return;
                }
                
                //Validation.
                this.msgSaveType = msgTypes.danger;
                if (this.password1 !== this.password2) {
                    this.msgSave = "Passwords do not match.";
                    return;
                }
                if (this.password1.length < this.minPasswordLength) {
                    this.msgSave = "The password you provided is too short. It must be at least " + this.minPasswordLength + " characters long.";
                    return;
                }
    
                //Validation ok.
                this.msgSave =      "Adding...";
                this.msgSaveType =  msgTypes.primary;
                this.submitting =   true;
    
                //Save password to object.
                this.userData.PasswordInput1 = this.password1;
                this.userData.PasswordInput2 = this.password2;
    
                //Perform api call.
                let data: Object = {
                    data: JSON.stringify(this.userData),
                };
                fetch(post(this.urls.add, data))
                .then(handleRequestErrors)
                .then(getJSON)
                .then(function (j) {
                    //check if response is an error from the server
                    let err: string = handleAPIErrors(j);
                    if (err !== '') {
                        manageUsers.msgSave =       err;
                        manageUsers.msgSaveType =   msgTypes.danger;
                        manageUsers.submitting =    false;
                        return;
                    }
    
                    //Refresh the list of users.
                    manageUsers.getUsers();
    
                    //Show success and reset the form
                    manageUsers.msgSave =       "Added!";
                    manageUsers.msgSaveType =   msgTypes.success;
                    setTimeout(function () {
                        manageUsers.resetUserData();
                        setToggle('btn-group-toggle', false, true);
                        manageUsers.msgSave =       '';
                        manageUsers.msgSaveType =   '';
                        manageUsers.submitting =    false;
                    }, defaultTimeout);
    
                    return;
                })
                .catch(function (err) {
                    console.log("fetch() error: >>", err, "<<");
                    manageUsers.msgSave =       'An unknown error occured.  Please try again.';
                    manageUsers.msgSaveType =   msgTypes.danger;
                    manageUsers.submitting =    false;
                    return;
                });
    
                return;
            },
    
            //addOrUpdate performs the correct action.
            addOrUpdate: function () {
                //Make sure data isn't already being submitted.
                if (this.submitting) {
                    console.log("already submitting...");                
                    return;
                }
                
                //Validation.
                //This validation is the same for updating or adding.
                this.msgSaveType = msgTypes.danger;
                if (this.userData.Username === '' || this.userData.Username === undefined) {
                    this.msgSave = "You must provide a username (email).";
                    return;
                }
                if (isEmail(this.userData.Username) === false) {
                    this.msgSave = "You must provide an email for a username";
                    return;
                }
    
                if (this.addingNew) {
                    this.add();
                }
                else {
                    this.update();
                }
    
                return;
            },
    
            //forceLogout sends the request to update the db to force a user to be 
            //logged out of the app. This works by having the db update a value that 
            //is checked every time the user is authenticated in the middleware which 
            //occurs on every page load or endpoint visit.
            //
            //This is useful for times when you want to make sure a user is logged out
            //for security purposes, updating something, etc.
            forceLogout: function() {
                if (this.forceLogoutSubmitting) {
                    return;
                }
    
                this.forceLogoutSubmitting =    true;
                this.forceLogoutBtnText =       "Forcing logout...";
    
                //Perform api call.
                let data: Object = {
                    userID: this.userData.ID,
                };
                fetch(post(this.urls.forceLogout, data))
                .then(handleRequestErrors)
                .then(getJSON)
                .then(function (j) {
                    //check if response is an error from the server
                    let err: string = handleAPIErrors(j);
                    if (err !== '') {
                        manageUsers.msgSave = err;
                        manageUsers.msgSaveType = msgTypes.danger;
                        return;
                    }
                    
                    //Make sure no errors are being displayed.
                    manageUsers.msgSave = '';
                    
                    //Show success.
                    manageUsers.forceLogoutBtnText = "Forcing logout...done!";
                    setTimeout(function() {
                        manageUsers.forceLogoutBtnText =    "Force Logout";
                        manageUsers.forceLogoutSubmitting = false;
                    }, defaultTimeout);
    
                    return;
                })
                .catch(function (err) {
                    console.log("fetch() error: >>", err, "<<");
                    manageUsers.msgSave =           'An unknown error occured.  Please try again.';
                    manageUsers.msgSaveType =       msgTypes.danger;
                    this.forceLogoutSubmitting =    false;
                    return;
                });
    
                return;
            },

            //resetChangePasswordModal calls the modalChangePassword.resetModal 
            //function to clear the inputs in the modal. This function is called when 
            //user clicks button top change a users password and open modal. This is 
            //done so that inputs are always clear when modal opens.
            resetChangePasswordModal: function() {
                modalChangePassword.resetModal();
                return;
            },
    
            //reset2FAActivationModal calls the resetModal function to clear any 
            //previous 2fa details from the modal to activate 2fa.
            reset2FAActivationModal: function() {
                modalActivate2FA.resetModal();
                return;
            },
        },
        mounted() {
            //Get the list of users.
            this.getUsers();

            //Get min password length from app settings saved in hidden input and used
            //when saving new user.
            let minPwdLen: string =     (document.getElementById("minPasswordLength") as HTMLInputElement).value;
            this.minPasswordLength =    parseInt(minPwdLen);
    
            return;
        }
    });
}

if (document.getElementById("modal-changePassword")) {
    //modalChangePassword handles the modal for setting a new password for a user
    //@ts-ignore cannot find name Vue
    var modalChangePassword = new Vue({
        name: 'modalChangePassword',
        delimiters: ['[[', ']]'],
        el: '#modal-changePassword',
        data: {
            //userID is the user that is being looked up/edited.
            //This is set in manageUsers.showUser() when a user is chosen.
            userID: 0,

            minPasswordLength: 10, //set in mounted by reading hidden input
    
            //things being edited
            password1: '',
            password2: '',
    
            //errors
            submitting: false,
            msg:        '',
            msgType:    '',

            //endpoints
            urls: {
                changePassword: "api/users/change-password/",
            },
        },
        methods: {
            //setUserData saves the provided id in this vue object. This is called 
            //any time a user is chosen in the "lookup" card. This can also be used 
            //hide this card in the UI if the userID is set to 0.
            /**
             * @param userID - The formulation id, > 0.
             */
            setUserData: function(userID: number) {
                this.userID = userID;
                return;
            },
    
            //save sets the new password.
            save: function() {
                //Make sure data isn't already being submitted.
                if (this.submitting) {
                    console.log("already submitting...");                
                    return;
                }
                
                //Validation.
                this.msgType = msgTypes.danger;
                if (this.userID < 1) {
                    this.msg = 'Cannot determine what user you want to set a new password for. Please refresh this page.';
                    return
                }
                if (this.password1 !== this.password2) {
                    this.msg = "Passwords do not match.";
                    return;
                }
                if (this.password1.length < this.minPasswordLength) {
                    this.msg = "The password you provided is too short. It must be at least " + this.minPasswordLength + " characters long.";
                    return;
                }
    
                //Validation ok.
                this.msgType =      msgTypes.primary;
                this.msg =          'Saving new password';
                this.submitting =   true;
    
                //Peform api call.
                let data: Object = {
                    userID:     this.userID,
                    password1:  this.password1,
                    password2:  this.password2,
                };
                fetch(post(this.urls.changePassword, data))
                .then(handleRequestErrors)
                .then(getJSON)
                .then(function (j) {
                    //check if response is an error from the server
                    let err: string = handleAPIErrors(j);
                    if (err !== '') {
                        modalChangePassword.msg =           err;
                        modalChangePassword.msgType =       msgTypes.danger;
                        modalChangePassword.submitting =    false;
                        return;
                    }
    
                    //show success and reset the form
                    modalChangePassword.msg =       "Password Updated!";
                    modalChangePassword.msgType =   msgTypes.success;
                    modalChangePassword.password1 = '';
                    modalChangePassword.password2 = '';
                    setTimeout(function() {
                        modalChangePassword.msg =           '';
                        modalChangePassword.submitting =    false;
                    }, defaultTimeout);
    
                    return;
                })
                .catch(function (err) {
                    console.log("fetch() error: >>", err, "<<");
                    modalChangePassword.msg =           'An unknown error occured.  Please try again.';
                    modalChangePassword.msgType =       msgTypes.danger;
                    modalChangePassword.submitting =    false;
                    return;
                });
    
                return;
            },

            //resetModal clears the inputs for setting a new password. This is called 
            //by the manageUsers.resetChangePasswordModal() when user clicks the 
            //button to open the modal.
            resetModal: function() {
                this.password1 =    "";
                this.password2 =    "";
                this.msg =          "";
                this.msgType =      "";
                this.submitting =   false;
                return;
            }
        },
        mounted() {
            //Get min password length from app settings saved in hidden input and used
            //when saving new password.
            let minPwdLen: string =     (document.getElementById("minPasswordLength") as HTMLInputElement).value;
            this.minPasswordLength =    parseInt(minPwdLen);
            return;
        }
    });
}

if (document.getElementById("modal-activate2FA")) {
    //modalActivate2FA handles the modal for activating 2 factor auth
    //@ts-ignore cannot find name Vue
    var modalActivate2FA = new Vue({
        name: 'modalActivate2FA',
        delimiters: ['[[', ']]'],
        el: '#modal-activate2FA',
        data: {
            //userID is the user that is being looked up/edited.
            //This is set in manageUsers.showUser() when a user is chosen.
            userID: 0,
    
            //stuff used to handle setting up 2FA
            twoFABarcode:           '', //store as base64 png
            twoFAVerificationCode:  '', //a 6 digit number but stored as text to make sure leading zeros aren't removed
            show2FAInfoOnly:        true, //show info, not the enrollment QR code.  so user can understand what 2fa is.
            retrievingBarcode:      false, //true when making request to get the enrollment qr code
    
            //errors
            submitting: false, 
            msg:        '',
            msgType:    '',

            //endpoints
            urls: {
                getQRCode: "api/users/2fa/get-qr-code/",
                verify:    "api/users/2fa/verify/",
            }
        },
        methods: {
            //setUserData saves the provided id in this vue object. This is called 
            //any time a user is chosen in the "lookup" card. This can also be used 
            //hide this card in the UI if the userID is set to 0.
            /**
             * @param userID - The user id, > 0.
             */
             setUserData: function(userID: number) {
                this.userID = userID;
                return;
            },
    
            //getBarcode retrieves the enrollment QR code for this user if the user is 
            //not currently enrolled in 2 factor auth. This returns a base64 string 
            //used as a the data attribute in an image tag in the html to display the 
            //QR code. This is easier than returning an actual png file.
            getBarcode: function() {
                //Make sure element to show barcode is showing.
                this.show2FAInfoOnly = false;
    
                //Perform api call.
                this.retrievingBarcode = true;
                let data: Object = {
                    userID: this.userID,
                };
                fetch(get(this.urls.getQRCode, data))
                .then(handleRequestErrors)
                .then(getJSON)
                .then(function (j) {
                    //check if response is an error from the server
                    let err: string = handleAPIErrors(j);
                    if (err !== '') {
                        modalActivate2FA.msg =                  err;
                        modalActivate2FA.msgType =              msgTypes.danger;
                        modalActivate2FA.retrievingBarcode =    false;
                        return;
                    }
    
                    //Put the barcode in the img tag.
                    modalActivate2FA.twoFABarcode =         j.Data;
                    modalActivate2FA.retrievingBarcode =    false;
                    return;
                })
                .catch(function (err) {
                    console.log("fetch() error: >>", err, "<<");
                    modalActivate2FA.msg =                  'An unknown error occured.  Please try again.';
                    modalActivate2FA.msgType =              msgTypes.danger;
                    modalActivate2FA.retrievingBarcode =    false;
                    return;
                });
    
                return;
            },
    
            //validate makes sure that the QR code a user scanned provided a valid 
            //2fa code. This basically performs the same work that providing a 2fa 
            //code upon login would do. If successful, this marks the user as fully 
            //activating 2fa.
            validate: function() {
                //Perform api call.
                this.submitting = true;
                let data: Object = {
                    userID:          this.userID,
                    validationCode:  this.twoFAVerificationCode,
                };
                fetch(post(this.urls.verify, data))
                .then(handleRequestErrors)
                .then(getJSON)
                .then(function (j) {
                    //check if response is an error from the server
                    let err: string = handleAPIErrors(j);
                    if (err !== '') {
                        modalActivate2FA.msg =          err;
                        modalActivate2FA.msgType =      msgTypes.danger;
                        modalActivate2FA.submitting =   false;
                        return;
                    }
    
                    //Success. Display success message that 2fa is now enabled for the 
                    //user.
                    modalActivate2FA.msg =      'Code verified! 2 Factor Authentication is now enabled for this user.';
                    modalActivate2FA.msgType =  msgTypes.success;
                   
                    //Mark user as having 2fa enabled.
                    for (let u of (manageUsers.users as user[])) {
                        if (u.ID === modalActivate2FA.userID) {
                            u.TwoFactorAuthEnabled = true;
                            break;
                        }
                    }
                      
                    //Never un-disabled "validate" button after 2fa has been verified. 
                    //User needs to reopen modal. Why? b/c validation never needs to 
                    //happen more than once.

                    return;
                })
                .catch(function (err) {
                    console.log("fetch() error: >>", err, "<<");
                    modalActivate2FA.msg = 'An unknown error occured.  Please try again.';
                    modalActivate2FA.msgType = msgTypes.danger;
                    modalActivate2FA.submitting = false;
                    return;
                });
    
                return;
            },

            //resetModal is called when the 2FA modal is launched. This sets the modal 
            //to the state for starting 2FA activation. This is used b/c the modal is 
            //left in the "success" state following a previous user activating 2FA so 
            //that the modal shows "success" until it is closed.
            resetModal: function() {
                this.msg =                      '';
                this.msgType =                  '';
                this.twoFABarcode =             '';
                this.twoFAVerificationCode =    '';
                this.show2FAInfoOnly =          true;
                this.submitting =               false;
                return;
            },

        }
    });
}

if (document.getElementById("modal-deactivate2FA")) {
    //modalDeactivate2FA handles the modal for deactivating 2 factor auth
    //@ts-ignore cannot find name Vue
    var modalDeactivate2FA = new Vue({
        name: 'modalDeactivate2FA',
        delimiters: ['[[', ']]'],
        el: '#modal-deactivate2FA',
        data: {
            //userID is the user that is being looked up/edited.
            //This is set in manageUsers.showUser() when a user is chosen.
            userID: 0,
    
            //errors
            submitting: false, 
            msg:        '',
            msgType:    '',

            //endpoints
            urls: {
                deactivate: "api/users/2fa/deactivate/",
            }
        },
        methods: {
            //setUserData saves the provided id in this vue object. This is called 
            //any time a user is chosen in the "lookup" card. This can also be used 
            //hide this card in the UI if the userID is set to 0.
            /**
             * @param userID - The user id, > 0.
             */
             setUserData: function(userID: number) {
                this.userID = userID;
                return;
            },
    
            //deactivate turns off 2fa for a user.
            deactivate: function() {
                //Perform api call.
                this.submitting = true;
                let data: Object = {
                    userID: this.userID,
                };
                fetch(post(this.urls.deactivate, data))
                .then(handleRequestErrors)
                .then(getJSON)
                .then(function (j) {
                    //check if response is an error from the server
                    let err: string = handleAPIErrors(j);
                    if (err !== '') {
                        modalDeactivate2FA.msg =        err;
                        modalDeactivate2FA.msgType =    msgTypes.danger;
                        modalDeactivate2FA.submitting = false;
                        return;
                    }
    
                    //Success. Display success message that 2fa is now disabled for 
                    //this user. Mark user as having 2fa disabled.
                    modalDeactivate2FA.msg = '2 Factor Authentication is now disabled for this user.';
                    modalDeactivate2FA.msgType = msgTypes.success;
                    setTimeout(function() {
                        for (let u of (manageUsers.users as user[])) {
                            if (u.ID === modalDeactivate2FA.userID) {
                                u.TwoFactorAuthEnabled = false;
                                break;
                            }
                        }
    
                        modalDeactivate2FA.msg =        '';
                        modalDeactivate2FA.msgType =    '';
                        modalDeactivate2FA.submitting = false;
                    }, defaultTimeout);
    
                    return;
                })
                .catch(function (err) {
                    console.log("fetch() error: >>", err, "<<");
                    modalDeactivate2FA.msg =        'An unknown error occured.  Please try again.';
                    modalDeactivate2FA.msgType =    msgTypes.danger;
                    modalDeactivate2FA.submitting = false;
                    return;
                });
    
                return;
            },
        }
    });
}
