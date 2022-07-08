/**
 * login.ts
 * The code in this file handles user login.
 * 
 * 2 factor authentication is handled like so: The user provides username
 * and password.  This is submitted to server which checks validity and if
 * 2FA is turned on.  If it is, the server checks if user has provided 2FA
 * code for this browser recently.  If yes, user is logged in.  If no, user
 * is shown 2FA code input.  The user then provided code and all info is
 * submitted again which is again checked for validity.
 * 
 * Login with bad password or bas 2FA code adds latency to reduce brute force
 * attempts of a password or code.
 */

/// <reference path="common.ts" />
/// <reference path="fetch.ts" />

if (document.getElementById("login")) {
    //@ts-ignore cannot find name Vue
    var login = new Vue({
        name: 'login',
        delimiters: ['[[', ']]'],
        el: '#login',
        data: {
            username: 'admin@example.com', //TODO: remove default, just used for quicker development
            password: 'admin@example.com',
            
            //errors
            msg:        '',
            msgType:    '',
            submitting: false,

            //2fa
            show2FAInput:           false,
            twoFAVerificationCode:  '',

            //endpoints
            urls: {
                loginAuth: "/login/",
                mainApp:   "/app/",
            }
        },
        methods: {
            login: function() {
                //make sure user isn't already trying to log in
                if (this.submitting) {
                    console.log("already submitting...");                
                    return;
                }
    
                //validation
                if (this.username.length === 0) {
                    return;
                }
                if (isEmail(this.username) === false) {
                    this.msg =      'You must provide an email address as a username.';
                    this.msgType =  msgTypes.danger;
                    return;
                }
                if (this.password.length === 0) {
                    this.msg =      'You must provide a password.';
                    this.msgType =  msgTypes.danger;
                    return;
                }
                if (this.show2FAInput && this.twoFAVerificationCode.length !== 6) {
                    this.msg =      'Your 2FA code must be exactly 6 numeric characters.';
                    this.msgType =  msgTypes.danger;
                    return
                }
    
                //validation ok
                this.msg =          'Logging in...';
                this.msgType =      msgTypes.primary;
                this.submitting =   true;
    
                //make request to try logging user in
                //if this is successful, session data will be set and we will redirect
                //user to main logged in page.
                let data: Object = {
                    username:   this.username,
                    password:   this.password,
                    twoFAToken: this.twoFAVerificationCode,
                };
                fetch(post(this.urls.loginAuth, data))
                .then(handleRequestErrors)
                .then(getJSON)
                .then(function(j) {
                    //check if response is an error from the server
                    let err: string = handleAPIErrors(j);
                    if (err !== '') {
                        login.msg =     err;
                        login.msgType = msgTypes.danger;
                        return;
                    }
    
                    //check response type to check 2fa status
                    let resType: string = j.Type;
                    if (resType === "login2FATokenRequired") {
                        //show user 2fa login ui
                        login.show2FAInput = true;
                        login.msg = '';
                        
                        //@ts-ignore
                        Vue.nextTick(function() {
                            document.getElementById('token').focus();
                        });

                        return;
                    }
                    else if (resType === "login2FAForced") {
                        //show user error message that they need to have admin set up 2fa for them
                        //2fa is required
                        login.msg = "2 Factor Authentication (2FA) is required but you do not have it set up.  Please see an administator to set up 2FA.";
                        login.msgType = msgTypes.danger;
                        return;
                    } 
                    else if  (resType === "loginOK") {
                        //login only required username/password and was successful
                        //or login also required 2fa and was successful
                        //redirect user to logged in page
                        window.location.href = login.urls.mainApp;
                        return;
                    }
                    else {
                        console.log("unhandled or invalid login response:", resType);
                        login.msg =     "An unknown error occured.  Please contact an administator.";
                        login.msgType = msgTypes.danger;
                        return;
                    }
                })
                .catch(function (err) {
                    console.log("fetch() error: >>", err, "<<");
                    login.msg =     'An unknown error occured.  Please try again.';
                    login.msgType = msgTypes.danger;
                    return;
                });
                
                this.submitting = false;
                return;
            }
        },
        mounted() {
            if (this.$el.id !== "" && document.getElementById(this.$el.id)) {
                //set cursor to username input on page load
                document.getElementById('username').focus();
            }
    
            return;
        }
    });
}
