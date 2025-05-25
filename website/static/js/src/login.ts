/**
 * login.ts
 * 
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

import { createApp, nextTick } from "vue";
import { isEmail, msgTypes } from "./common";
import { post, handleRequestErrors, getJSON, handleAPIErrors } from "./fetch";

if (document.getElementById("login")) {
    const login = createApp({
        name: "login",

        compilerOptions: {
            delimiters: ["[[", "]]"],
        },

        data() {
            return {
                username: "",
                password: "",

                //Errors.
                msg: "",
                msgType: "",
                submitting: false,

                //2FA stuff.
                show2FAInput: false,
                twoFAVerificationCode: "",

                //Endpoints.
                urls: {
                    loginAuth: "/app/login/",

                    mainApp: "/app/", //redirect to page.
                }
            }
        },

        methods: {
            login() {
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
                    this.msg = "You must provide an email address as a username.";
                    this.msgType = msgTypes.danger;
                    return;
                }
                if (this.password.length === 0) {
                    this.msg = "You must provide a password.";
                    this.msgType = msgTypes.danger;
                    return;
                }
                if (this.show2FAInput && this.twoFAVerificationCode.length !== 6) {
                    this.msg = "Your 2FA code must be exactly 6 numeric characters.";
                    this.msgType = msgTypes.danger;
                    return
                }

                //Make API request.
                //
                //If this is successful, session data will be set in cookie by response 
                //and we will redirect user to the main logged in page.
                this.msg = "Logging in...";
                this.msgType = msgTypes.primary;
                this.submitting = true;

                let data: Object = {
                    username: this.username,
                    password: this.password,
                    twoFAToken: this.twoFAVerificationCode,
                };
                fetch(post(this.urls.loginAuth, data))
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

                        //Check response type to check 2fa status. This handles showing
                        //the 2fa token input if needed.
                        let resType: string = j.Type;
                        if (resType === "login2FATokenRequired") {
                            //Show user 2fa token input.
                            this.show2FAInput = true;
                            this.msg = "";
                            this.submitting = false;

                            nextTick(function () {
                                (document.getElementById("token") as HTMLElement).focus();
                            });

                            return;
                        }
                        else if (resType === "login2FAForced") {
                            //Show user error message that they need to have admin set up 
                            //2fa for them, 2fa is required and this user does not have
                            //2fa set up yet.
                            this.msg = "2 Factor Authentication (2FA) is required but you do not have it set up. Please see an administator to set up 2FA.";
                            this.msgType = msgTypes.danger;
                            this.submitting = false;
                            return;
                        }
                        else if (resType === "loginOK") {
                            //Login was successful. Either user only had to provide username
                            //and password (2fa is not enabled for this user or 2fa token
                            //was provided recently) or user provided username, password, 
                            //and 2fa token.
                            //
                            //Redirect user to main logged in page.
                            window.location.href = this.urls.mainApp;
                            return;
                        }
                        else {
                            console.log("unhandled or invalid login response:", resType);
                            this.msg = "An unknown error occured. Please contact an administator.";
                            this.msgType = msgTypes.danger;
                            this.submitting = false;
                            return;
                        }
                    })
                    .catch((err) => {
                        console.log("fetch() error: >>", err, "<<");
                        this.msg = "An unknown error occured. Please try again.";
                        this.msgType = msgTypes.danger;
                        this.submitting = false;
                        return;
                    });

                return;
            }
        },

        mounted() {
            //Set cursor to username input on page load.
            let usernameElem = document.getElementById('username');
            if (usernameElem) {
                usernameElem.focus();
            }
            return;
        }
    }).mount("#login");
}
