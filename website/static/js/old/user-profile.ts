/**
 * user-profile.ts
 * The code in this is used user profile page.
 */

/// <reference path="common.ts" />
/// <reference path="fetch.ts" />
/// <reference path="types.ts" />

if (document.getElementById("userProfile")) {
    //@ts-ignore cannot find name Vue
    var userProfile = new Vue({
        name: 'userProfile',
        delimiters: ['[[', ']]'],
        el: '#userProfile',
        data: {
            //Errors.
            msg: '',
            msgType: '',

            //User's data, retrieved in mounted().
            userData: {} as user,
            userDataRetrieved: false,

            //Endpoints.
            urls: {
                getUserData: "/api/user/",
            }
        },
        methods: {
            //getUserData looks up the data for the currently logged in user.
            getUserData: function () {
                let data: Object = {};
                fetch(get(this.urls.getUserData, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then(function (j) {
                        //Check if response is an error from the server.
                        let err: string = handleAPIErrors(j);
                        if (err !== '') {
                            userProfile.msg = err;
                            userProfile.msgType = msgTypes.danger;
                            return;
                        }

                        //Save data for building GUI with.
                        userProfile.userData = j.Data || [];
                        userProfile.userDataRetrieved = true;

                        //Pass user ID to other Vue objects for handling changing
                        //password and 2FA stuff.
                        userProfile.populateUserIDInOtherVueObjects();

                        return;
                    })
                    .catch(function (err) {
                        console.log("fetch() error: >>", err, "<<");
                        userProfile.msg = 'An unknown error occured. Please try again.';
                        userProfile.msgType = msgTypes.danger;
                        return;
                    });

                return;
            },

            //populateUserIDInOtherVueObjects sets the user's ID in other Vue objects
            //for handling modals.
            populateUserIDInOtherVueObjects: function () {
                modalChangePassword.setUserData(this.userData.ID);
                modalActivate2FA.setUserData(this.userData.ID);
                modalDeactivate2FA.setUserData(this.userData.ID);
            },

            //resetChangePasswordModal calls the modalChangePassword.resetModal 
            //function to clear the inputs in the modal. This function is called when 
            //user clicks button top change a users password and open modal. This is 
            //done so that inputs are always clear when modal opens.
            resetChangePasswordModal: function () {
                modalChangePassword.resetModal();
                return;
            },

            //reset2FAActivationModal calls the resetModal function to clear any 
            //previous 2fa details from the modal to activate 2fa.
            reset2FAActivationModal: function () {
                modalActivate2FA.resetModal();
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
    });
}

//Change password, activate 2FA, and deactivate 2FA are handled via the Vue objects
//in users.ts since the code is the same.