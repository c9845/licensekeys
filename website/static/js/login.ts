/**
 * login.ts
 * The code in this file handles user login.
 * 
 * 2 factor authentication is handled like so: The user provides username and password. 
 * This is submitted to server which checks if both are valid and if 2FA is turned on. 
 * If it is, the server checks if user has provided 2FA token for this browser 
 * recently (id stored in cookie and verified in database). If yes, user is logged in. 
 * If no, user is shown 2FA token input. The user then provides the token and all info 
 * is submitted again which is again (username and password plus 2FA token).
 * 
 * Login with bad password or bad 2FA code adds latency to reduce brute force attempts 
 * of a password or code.
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
            username: '',
            password: '',
            
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
                //Make sure user isn't already trying to log in.
                if (this.submitting) {
                    console.log("already submitting...");                
                    return;
                }
    
                //Validation.
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
    
                //Validation ok.
                this.msg =          'Logging in...';
                this.msgType =      msgTypes.primary;
                this.submitting =   true;
    
                //Make request to try logging user in. If this is successful, session 
                //data will be set in cookie by response and we will redirect user to 
                //the main logged in page.
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
    
                    //Check response type to check 2fa status. This handles showing
                    //the 2fa token input if needed.
                    let resType: string = j.Type;
                    if (resType === "login2FATokenRequired") {
                        //Show user 2fa token input.
                        login.show2FAInput = true;
                        login.msg = '';
                        
                        //@ts-ignore
                        Vue.nextTick(function() {
                            document.getElementById('token').focus();
                        });

                        return;
                    }
                    else if (resType === "login2FAForced") {
                        //Show user error message that they need to have admin set up 
                        //2fa for them, 2fa is required and this user does not have
                        //2fa set up yet.
                        login.msg =     "2 Factor Authentication (2FA) is required but you do not have it set up.  Please see an administator to set up 2FA.";
                        login.msgType = msgTypes.danger;
                        return;
                    } 
                    else if  (resType === "loginOK") {
                        //Login was successful. Either user only had to provide username
                        //and password (2fa is not enabled for this user or 2fa token
                        //was provided recently) or user provided username, password, 
                        //and 2fa token.
                        //
                        //Redirect user to main logged in page.
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
            //Set cursor to username input on page load.
            document.getElementById('username').focus();
            return;
        }
    });
}
