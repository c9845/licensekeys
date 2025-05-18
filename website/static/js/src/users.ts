/**
 * users.ts
 * 
 * This code is used on the user management page. This allows management of users, as 
 * well as adding users.
 * 
 * This code is also reused on the user-profile page, where a logged in user can
 * update their password and 2FA settings. The user-management and user-profile pages
 * use a lot of the same components (change password, 2FA stuff) so it just made
 * sense storing all this data together.
 */

import { createApp } from "vue";
import { Modal } from "bootstrap";
import { isEmail, msgTypes, apiBaseURL, defaultTimeout } from "./common";
import { get, post, handleRequestErrors, getJSON, handleAPIErrors } from "./fetch";

//Manage the list of users. Update or add users.
var manageUsers: any; //must be "any", not "ComponentPublicInstance" to remove errors when calling functions (methods) of this Vue instance.
if (document.getElementById("manageUsers")) {
    manageUsers = createApp({
        name: 'manageUsers',

        compilerOptions: {
            delimiters: ["[[", "]]"],
        },

        data() {
            return {
                //Handle GUI states:
                // - page just loaded and no user selected: show lookup select inputs.
                // - user wants to add user: show add/edit inputs.
                // - user chose a user: show lookup select inputs & add/edit inputs.
                addingNew: false,

                //List of users.
                users: [] as user[],
                usersRetrieved: false,
                msgLoad: "",
                msgLoadType: "",

                //Single user selected.
                userSelectedID: 0,
                userData: {
                    Active: true, //New users are always active, because why would you create a new user if they are inactive?
                } as user,

                //Form submission stuff.
                submitting: false,
                msgSave: "",
                msgSaveType: "",

                //Misc.
                minPasswordLength: 10, //set in mounted by reading hidden input, based on config file value.

                //Hide/show details.
                hidePermissionDescriptions: true,

                //Endpoints.
                urls: {
                    getUsers: apiBaseURL + "users/",
                    update: apiBaseURL + "users/update/",
                    add: apiBaseURL + "users/add/",
                }
            }
        },

        computed: {
            //addEditCardTitle sets the text of the card used for adding or editing.
            //Since the card used for adding & editing is the same we want to show the 
            //correct card title to the user so they know what they are doing.
            addEditCardTitle: function () {
                if (this.addingNew) {
                    return "Add User";
                }

                return "Edit User";
            },
        },

        methods: {
            //getUsers gets the list of users.
            getUsers: function () {
                let data: Object = {};
                fetch(get(this.urls.getUsers, data))
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

                        this.users = j.Data || [];
                        this.usersRetrieved = true;
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

            //setState handles setting the GUI state to the "add" or "lookup/edit" 
            //state.
            //
            //This is called when a user clicks the add or view buttons in the card
            //header or when a new user is saved.
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
            },

            //resetForm resets the inputs and toggles when showing the "add user"
            //GUI, or after a user is added and we empty all the inputs.
            resetForm: function () {
                this.userData = {
                    Username: "",
                    Fname: "",
                    Lname: "",

                    Active: true, //new users are always active.
                    Administrator: false,

                    CreateLicenses: false,
                    ViewLicenses: false,

                    PasswordInput1: "",
                    PasswordInput2: "",
                } as user;
                this.userSelectedID = 0;

                return;
            },

            //showUser populates the "lookup" GUI with data about a user chosen from 
            //the select menu.
            showUser: function () {
                //Make sure a user was selected.
                if (this.userSelectedID < 1) {
                    return;
                }

                //Get user's data from the list of users we retrieved.
                for (let u of (this.users as user[])) {
                    if (u.ID !== this.userSelectedID) {
                        continue;
                    }

                    //Save the chosen user for displaying in the GUI.
                    this.userData = u;
                    break;
                }

                //Set the user ID in other Vue objects.
                this.setUserIDInOtherVueObjects(this.userSelectedID);

                return;
            },

            //setUserIDInOtherVueObjects passes the user ID from the chosen user to
            //other Vue objects that manage specific tasks (update password, etc.)
            //
            //This function is called in showUser() when a user is chosen from the
            //select menu list of users.
            setUserIDInOtherVueObjects(userID: number) {
                modalChangePassword.setUserData(userID);
                modalActivate2FA.setUserData(userID);
                modalDeactivate2FA.setUserData(userID);
                modalForceLogout.setUserData(userID);
                return;
            },

            //update saves changes made to a user.
            update: function () {
                //Validation.
                if (isNaN(this.userData.ID) || this.userData.ID === "" || this.userData.ID < 1) {
                    this.msgSave = "An error occurred. Please reload this page and try again.";
                    this.msgSaveType = msgTypes.danger;
                    return;
                }

                //Make API request.
                this.msgSave = "Saving...";
                this.msgSaveType = msgTypes.primary;
                this.submitting = true;

                let data: Object = {
                    data: JSON.stringify(this.userData),
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
                        this.msgSave = "An unknown error occurred. Please try again.";
                        this.msgSaveType = msgTypes.danger;
                        this.submitting = false;
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
                if (this.userData.PasswordInput1 !== this.userData.PasswordInput2) {
                    this.msgSave = "Passwords do not match.";
                    return;
                }
                if (this.userData.PasswordInput1.length < this.minPasswordLength) {
                    this.msgSave = "The password you provided is too short. It must be at least " + this.minPasswordLength + " characters long.";
                    return;
                }

                //Make API request.
                this.msgSave = "Adding...";
                this.msgSaveType = msgTypes.primary;
                this.submitting = true;

                let data: Object = {
                    data: JSON.stringify(this.userData),
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

                        //Refresh the list of users.
                        this.getUsers();

                        //Show success and reset the form.
                        this.msgSave = "User added!";
                        this.msgSaveType = msgTypes.success;

                        setTimeout(() => {
                            this.resetForm();
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
                if (this.userData.Fname === "") {
                    this.msgSave = "You must provide the user's first name.";
                    return;
                }
                if (this.userData.Lname === "") {
                    this.msgSave = "You must provide the user's last name.";
                    return;
                }
                if (this.userData.Username === "" || this.userData.Username === undefined) {
                    this.msgSave = "You must provide a username (email).";
                    return;
                }
                if (isEmail(this.userData.Username) === false) {
                    this.msgSave = "You must provide an email for a username";
                    return;
                }

                //Perform correct task.
                if (this.addingNew) {
                    this.add();
                }
                else {
                    this.update();
                }

                return;
            },

            //resetChangePasswordModalForm calls the modalChangePassword.resetForm() 
            //function to clear the inputs in the modal that handles setting a new
            //password for a user. This is done so that the inputs for changing a 
            //password are always clear when the change password modal is opened.
            //
            //This function is called when a user clicks the button to launch the 
            //change password modal. 
            resetChangePasswordModalForm: function () {
                modalChangePassword.resetForm();
                return;
            },

            //reset2FAActivationModalForm calls the modalActivate2FA.resetForm() 
            //function to clear any previous details from the modal that handles
            //activating 2FA. This is done so that the inputs for activating 2FA are
            //always clear when the activate 2FA modal is opened.
            //
            //This function is called when a user clicks the button to launch the
            //activate 2FA modal.
            reset2FAActivationModalForm: function () {
                modalActivate2FA.resetForm();
                return;
            },

            //setTwoFactorAuthEnabled updates the TwoFactorAuthEnabled field for the
            //currently displayed user.
            //
            //This function is called from the Vue instances that handle the 2FA
            //enrollment and disabling modals.
            setTwoFactorAuthEnabled: function (b: boolean) {
                this.userData.TwoFactorAuthEnabled = b;

                for (let u of (this.users as user[])) {
                    if (u.ID === this.userSelectedID) {
                        u.TwoFactorAuthEnabled = b;
                        break;
                    }
                }

                return;
            }
        },

        mounted() {
            //Get the list of users.
            this.getUsers();

            //Get min password length from app settings saved in hidden input and used
            //when saving new user.
            let minPwdLen: string = (document.getElementById("minPasswordLength") as HTMLInputElement).value;
            this.minPasswordLength = parseInt(minPwdLen);

            return;
        }
    }).mount("#manageUsers");
}

//Modal to change a user's password.
var modalChangePassword: any; //must be "any", not "ComponentPublicInstance" to remove errors when calling functions (methods) of this Vue instance.
if (document.getElementById("modal-changePassword")) {
    modalChangePassword = createApp({
        name: "modalChangePassword",

        compilerOptions: {
            delimiters: ["[[", "]]"],
        },

        data() {
            return {
                //userID is the user that is being looked up/edited.
                //
                //This is set in setUserData() which is called from manageUsers.showUser() 
                //when a user is chosen.
                userID: 0,

                //Shortest password length we allow. The only password validation we do.
                //
                //This is set in mounted(), by reading a hidden input set by golang
                //templating based on value set in config file.
                minPasswordLength: 10,

                //Things being edited.
                password1: "",
                password2: "",

                //Form submission stuff.
                submitting: false,
                msg: "",
                msgType: "",

                //Endpoints.
                urls: {
                    changePassword: apiBaseURL + "users/change-password/",
                },
            }
        },

        methods: {
            //setUserData saves the provided user's ID in this Vue instance.
            //
            //This is called any time a user is chosen from the select menu in the "lookup" 
            //GUI.
            setUserData: function (userID: number) {
                this.userID = userID;
                return;
            },

            //save saves a new password.
            save: function () {
                //Make sure data isn't already being submitted.
                if (this.submitting) {
                    console.log("already submitting...");
                    return;
                }

                //Validation.
                this.msgType = msgTypes.danger;
                if (this.userID < 1) {
                    this.msg = "Cannot determine what user you want to set a new password for. Please refresh this page.";
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

                //Make API request.
                this.msgType = msgTypes.primary;
                this.msg = "Saving new password";
                this.submitting = true;

                let data: Object = {
                    userID: this.userID,
                    password1: this.password1,
                    password2: this.password2,
                };
                fetch(post(this.urls.changePassword, data))
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

                        //Show success and reset the form. Handle different error message
                        //if a user changed their own password so user knows they will be
                        //logged out.
                        let changedOwnPassword: boolean = j.Data.ChangedOwnPassword;
                        if (changedOwnPassword) {
                            this.msg = "Password Updated! You will now be logged out.";
                            this.msgType = msgTypes.success;
                            setTimeout(() => {
                                window.location.href = "/app/login/?status=password-updated";
                            }, defaultTimeout);
                        } else {
                            this.msg = "Password Updated!";
                            this.msgType = msgTypes.success;
                            this.password1 = "";
                            this.password2 = "";
                            setTimeout(() => {
                                this.msg = "";
                                this.submitting = false;
                            }, defaultTimeout);
                        }
                        return;
                    })
                    .catch((err) => {
                        console.log("fetch() error: >>", err, "<<");
                        this.msg = "An unknown error occurred. Please try again.";
                        this.msgType = msgTypes.danger;
                        this.submitting = false;
                        return;
                    });

                return;
            },

            //resetForm clears the inputs for setting a new password.
            // 
            //This is called by manageUsers.resetChangePasswordModalForm() and
            //userProfile.resetChangePasswordModalForm() when a user clicks the button 
            //to open the modal.
            resetForm: function () {
                this.password1 = "";
                this.password2 = "";
                this.msg = "";
                this.msgType = "";
                this.submitting = false;
                return;
            }
        },
        mounted() {
            //Get min password length from app settings saved in hidden input and used
            //when saving new password.
            let minPwdLen: string = (document.getElementById("minPasswordLength") as HTMLInputElement).value;
            this.minPasswordLength = parseInt(minPwdLen);
            return;
        }
    }).mount("#modal-changePassword");
}

//Modal to for a user to logout.
var modalForceLogout: any; //must be "any", not "ComponentPublicInstance" to remove errors when calling functions (methods) of this Vue instance.
if (document.getElementById("modal-forceLogout")) {
    modalForceLogout = createApp({
        name: "modalForceLogout",

        compilerOptions: {
            delimiters: ["[[", "]]"],
        },

        data() {
            return {
                //userID is the user that is being looked up/edited.
                //
                //This is set in setUserData() which is called from manageUsers.showUser() 
                //when a user is chosen.
                userID: 0,

                //Form submission stuff.
                submitting: false,
                msg: "",
                msgType: "",

                //Endpoints.
                urls: {
                    forceLogout: apiBaseURL + "users/force-logout/",
                }
            }
        },

        methods: {
            //setUserData saves the provided user's ID in this Vue instance.
            //
            //This is called any time a user is chosen from the select menu in "lookup" 
            //GUI.
            setUserData: function (userID: number) {
                this.userID = userID;
                return;
            },

            //forceLogout immediatly terminates all existing sessions for a user. A user
            //will have to log back in.
            forceLogout: function () {
                //Make API request.
                this.submitting = true;
                let data: Object = {
                    userID: this.userID,
                };
                fetch(post(this.urls.forceLogout, data))
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

                        //Success.
                        this.msg = "User has been logged out!";
                        this.msgType = msgTypes.success;

                        //Automatically close the modal since there isn't anything else
                        //for the user to do.
                        setTimeout(() => {
                            let elem: HTMLElement = document.getElementById("modal-forceLogout")!;
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

                return;
            },
        }
    }).mount("#modal-forceLogout");
}

//Modal to activate 2FA.
var modalActivate2FA: any; //must be "any", not "ComponentPublicInstance" to remove errors when calling functions (methods) of this Vue instance.
if (document.getElementById("modal-activate2FA")) {
    modalActivate2FA = createApp({
        name: "modalActivate2FA",

        compilerOptions: {
            delimiters: ["[[", "]]"],
        },

        data() {
            return {
                //userID is the user that is being looked up/edited.
                //
                //This is set in setUserData() which is called from manageUsers.showUser() 
                //when a user is chosen.
                userID: 0,

                //Stuff used to handle setting up 2FA.
                twoFABarcode: "", //store as base64 png
                twoFAVerificationCode: "", //a 6 digit number but stored as text to make sure leading zeros aren't removed
                show2FAInfoOnly: true, //show info, not the enrollment QR code.  so user can understand what 2fa is.
                retrievingBarcode: false, //true when making request to get the enrollment qr code

                //Form submission stuff.
                submitting: false,
                msg: "",
                msgType: "",

                //Endpoints.
                urls: {
                    getQRCode: apiBaseURL + "users/2fa/get-qr-code/",
                    verify: apiBaseURL + "users/2fa/verify/",
                },
            }
        },

        methods: {
            //setUserData saves the provided user's ID in this Vue instance.
            //
            //This is called any time a user is chosen from the select menu in "lookup" 
            //GUI.
            setUserData: function (userID: number) {
                this.userID = userID;
                return;
            },

            //getBarcode retrieves the enrollment QR code for this user if the user is 
            //not currently enrolled in 2 factor auth. This returns a base64 string 
            //used as a the data attribute in an <img>> tag in the HTML to display the 
            //QR code. This is easier than returning an actual png file.
            getBarcode: function () {
                //Make sure element to show barcode is showing.
                this.show2FAInfoOnly = false;

                //Make API request.
                this.retrievingBarcode = true;
                let data: Object = {
                    userID: this.userID,
                };
                fetch(get(this.urls.getQRCode, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then((j) => {
                        //Check if response is an error from the server.
                        let err: string = handleAPIErrors(j);
                        if (err !== "") {
                            this.msg = err;
                            this.msgType = msgTypes.danger;
                            this.retrievingBarcode = false;
                            return;
                        }

                        //Put the barcode in the <img> tag.
                        this.twoFABarcode = j.Data;
                        this.retrievingBarcode = false;
                        return;
                    })
                    .catch((err) => {
                        console.log("fetch() error: >>", err, "<<");
                        this.msg = "An unknown error occurred. Please try again.";
                        this.msgType = msgTypes.danger;
                        this.retrievingBarcode = false;
                        return;
                    });

                return;
            },

            //validate makes sure that the QR code a user scanned provided a valid 
            //2FA code. This basically performs the same work that providing a 2FA 
            //code upon login would do. If successful, this marks the user as fully 
            //activating 2FA.
            validate: function () {
                //Make sure a validation code was provided.
                if (this.twoFAVerificationCode.trim().length !== 6) {
                    this.msg = "The verification code you provided is invalid. The code must be 6 numbers long.";
                    this.msgType = msgTypes.danger;
                    return;
                }

                //Make API request.
                this.msg = "Verifying...";
                this.msgType = msgTypes.primary;
                this.submitting = true;

                let data: Object = {
                    userID: this.userID,
                    validationCode: this.twoFAVerificationCode,
                };
                fetch(post(this.urls.verify, data))
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

                        //Success. Display success message that 2FA is now enabled for the 
                        //user.
                        this.msg = "Code verified! 2 Factor Authentication is now enabled for this user.";
                        this.msgType = msgTypes.success;

                        //Mark user as having 2FA enabled. Have to handle list of users,
                        //for admin's working on Manage Users page, and single user, for
                        //user on User Profile page.
                        if (document.getElementById("manageUsers")) {
                            manageUsers.setTwoFactorAuthEnabled(true);
                        }
                        else if (document.getElementById("userProfile")) {
                            userProfile.userData.TwoFactorAuthEnabled = true;
                        }

                        //Never un-disabled "validate" button after 2FA has been verified. 
                        //User needs to reopen modal. Why? b/c validation never needs to 
                        //happen more than once.

                        return;
                    })
                    .catch((err) => {
                        console.log("fetch() error: >>", err, "<<");
                        this.msg = "An unknown error occurred. Please try again.";
                        this.msgType = msgTypes.danger;
                        this.submitting = false;
                        return;
                    });

                return;
            },

            //resetForm clears the inputs for enrolling 2FA.
            //
            //This is called by manageUsers.reset2FAActivationModalForm() and
            //userProfile.reset2FAActivationModalForm() when a user clicks the button
            //to open the modal.
            resetForm: function () {
                this.msg = "";
                this.msgType = "";
                this.twoFABarcode = "";
                this.twoFAVerificationCode = "";
                this.show2FAInfoOnly = true;
                this.submitting = false;
                return;
            },
        }
    }).mount("#modal-activate2FA");
}

//Modal to deactivate 2FA.
var modalDeactivate2FA: any; //must be "any", not "ComponentPublicInstance" to remove errors when calling functions (methods) of this Vue instance.
if (document.getElementById("modal-deactivate2FA")) {
    modalDeactivate2FA = createApp({
        name: "modalDeactivate2FA",

        compilerOptions: {
            delimiters: ["[[", "]]"],
        },

        data() {
            return {
                //userID is the user that is being looked up/edited.
                //
                //This is set in setUserData() which is called from manageUsers.showUser() 
                //when a user is chosen.
                userID: 0,

                //Form submission stuff.
                submitting: false,
                msg: "",
                msgType: "",

                //Endpoints.
                urls: {
                    deactivate: apiBaseURL + "users/2fa/deactivate/",
                }
            }
        },

        methods: {
            //setUserData saves the provided user's ID in this Vue instance.
            //
            //This is called any time a user is chosen from the select menu in "lookup" 
            //GUI.
            setUserData: function (userID: number) {
                this.userID = userID;
                return;
            },

            //deactivate turns off 2FA for a user.
            deactivate: function () {
                //Make API request.
                this.submitting = true;
                let data: Object = {
                    userID: this.userID,
                };
                fetch(post(this.urls.deactivate, data))
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

                        //Success. Display success message that 2FA is now disabled for 
                        //this user.
                        this.msg = "2 Factor Authentication is now disabled for this user.";
                        this.msgType = msgTypes.success;

                        //Mark user as having 2FA disabled. Have to handle list of users,
                        //for admin's working on Manage Users page, and single user, for
                        //user on User Profile page.
                        if (document.getElementById("manageUsers")) {
                            manageUsers.setTwoFactorAuthEnabled(false);
                        }
                        else if (document.getElementById("userProfile")) {
                            userProfile.userData.TwoFactorAuthEnabled = false;
                        }

                        //Automatically close the modal since there isn't anything else
                        //for the user to do.
                        setTimeout(() => {
                            let elem: HTMLElement = document.getElementById("modal-deactivate2FA")!;
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

                return;
            },
        }
    }).mount("#modal-deactivate2FA");
}

//***********************************************************************************

//User can manage their profile. This really only show's user's basic data and then
//handles passing data to allow a user to update their own password or manage their
//2FA.
var userProfile: any; //must be "any", not "ComponentPublicInstance" to remove errors when calling functions (methods) of this Vue instance.
if (document.getElementById("userProfile")) {
    userProfile = createApp({
        name: "userProfile",

        compilerOptions: {
            delimiters: ["[[", "]]"],
        },

        data() {
            return {
                //Errors.
                msg: '',
                msgType: '',

                //User's data, retrieved in mounted().
                userData: {} as user,
                userDataRetrieved: false,

                //Endpoints.
                urls: {
                    getUserData: apiBaseURL + "user/",
                }
            }
        },

        methods: {
            //getUserData looks up the data for the currently logged in user.
            getUserData: function () {
                let data: Object = {};
                fetch(get(this.urls.getUserData, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then((j) => {
                        //Check if response is an error from the server.
                        let err: string = handleAPIErrors(j);
                        if (err !== '') {
                            this.msg = err;
                            this.msgType = msgTypes.danger;
                            return;
                        }

                        //Save data for building GUI with.
                        this.userData = j.Data || [];
                        this.userDataRetrieved = true;

                        //Set the user ID in other Vue objects.
                        this.setUserIDInOtherVueObjects(this.userData.ID);

                        return;
                    })
                    .catch((err) => {
                        console.log("fetch() error: >>", err, "<<");
                        this.msg = 'An unknown error occurred. Please try again.';
                        this.msgType = msgTypes.danger;
                        return;
                    });

                return;
            },

            //setUserIDInOtherVueObjects passes the user ID from the chosen user to
            //other Vue objects that manage specific tasks (update password, etc.)
            //
            //This function is called in getUserData() upon page load.
            setUserIDInOtherVueObjects: function (userID: number) {
                modalChangePassword.setUserData(userID);
                modalActivate2FA.setUserData(userID);
                modalDeactivate2FA.setUserData(userID);
            },

            //resetChangePasswordModalForm calls the modalChangePassword.resetForm() 
            //function to clear the inputs in the modal that handles setting a new
            //password for a user. This is done so that the inputs for changing a 
            //password are always clear when the change password modal is opened.
            //
            //This function is called when a user clicks the button to launch the 
            //change password modal. 
            resetChangePasswordModalForm: function () {
                modalChangePassword.resetForm();
                return;
            },

            //reset2FAActivationModalForm calls the modalActivate2FA.resetForm() 
            //function to clear any previous details from the modal that handles
            //activating 2FA. This is done so that the inputs for activating 2FA are
            //always clear when the activate 2FA modal is opened.
            //
            //This function is called when a user clicks the button to launch the
            //activate 2FA modal.
            reset2FAActivationModalForm: function () {
                modalActivate2FA.resetForm();
                return;
            },
        },

        mounted() {
            //Get the data for the logged in user to build GUI with.
            this.getUserData();

            //Get min password length from app settings saved in hidden input and used
            //when saving new user.
            let minPwdLen: string = (document.getElementById("minPasswordLength") as HTMLInputElement).value;
            this.minPasswordLength = parseInt(minPwdLen);

            return;
        }
    }).mount("#userProfile");
}
