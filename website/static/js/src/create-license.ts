/**
 * create-license.ts
 * 
 * This file handles creating a new license. The user picks an app and a keypair,
 * then provides the common fields and custom fields, if needed, and the license
 * is created and signed using the keypair. The user is directed to a page for
 * managing the license and downloading the license file.
 */

import { createApp } from "vue";
import { msgTypes, apiBaseURL, defaultTimeout, todayPlus, getToday, isEmail } from "./common";
import { get, post, handleRequestErrors, getJSON, handleAPIErrors } from "./fetch";

if (document.getElementById("createLicense")) {
    const createLicense = createApp({
        name: 'createLicense',

        compilerOptions: {
            delimiters: ['[[', ']]'],
        },

        data() {
            return {
                //Apps a license can be created for.
                apps: [] as app[],
                appsRetrieved: false,
                appSelectedID: 0,

                //Keypairs for chosen app.
                keyPairs: [] as keyPair[],
                keyPairsRetrieved: false,

                //Custom fields for chosen app.
                //
                //This will also store the value chosen for each field. We do this
                //since it would just be messy to store the field ID and value in
                //another array of objects. While this doesn't end up being a "defined"
                //field, a "defined" and "result" field are similar enough to do this.
                //We translate a defined field to a result when saving the license data.
                fields: [],
                fieldsRetrieved: false,

                //Need these for v-if in html.
                customFieldTypeInteger: customFieldTypeInteger,
                customFieldTypeDecimal: customFieldTypeDecimal,
                customFieldTypeText: customFieldTypeText,
                customFieldTypeBoolean: customFieldTypeBoolean,
                customFieldTypeMultiChoice: customFieldTypeMultiChoice,
                customFieldTypeDate: customFieldTypeDate,

                //Separator for multichoice options.
                separator: ";",

                //Data for license being created.
                licenseData: {
                    KeyPairID: 0, //From chosen/default keypair for app.
                    CompanyName: "",
                    ContactName: "",
                    PhoneNumber: "",
                    Email: "",
                    ExpireDate: "", //By default, this is set to "today" plus the app's DaysToExpiration.
                } as license,

                //Form submissing stuff.
                submitting: false,
                msg: '',
                msgType: '',

                //Endpoints.
                urls: {
                    getApps: apiBaseURL + "apps/",
                    getKeyPairs: apiBaseURL + "key-pairs/",
                    getCustomFields: apiBaseURL + "custom-fields/defined/",
                    add: apiBaseURL + "licenses/add/",
                    getAPIKeys: apiBaseURL + "api-keys/",
                },

                //Used for displaying the API builder. This shows a GUI of the API
                //request a user would need to build to create a license via the API.
                showAPIBuilder: false,
                returnLicenseFile: false,
                apiKeys: [] as apiKey[],
                apiKeysRetrieved: false,
                apiBuilderMsg: "",
                apiBuilderMsgType: "",
                apiKeySelected: "", //not the ID, but the actual key
                apiExample: "", //the example curl request to show in the GUI.
            }
        },

        methods: {
            //getApps gets the list of apps that have been defined.
            getApps: function () {
                let data: Object = {};
                fetch(get(this.urls.getApps, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then((j) => {
                        //Check if response is an error from the server.
                        let err: string = handleAPIErrors(j);
                        if (err !== "") {
                            this.msg = err;
                            this.msgType = msgTypes.danger;
                            return;
                        }

                        this.apps = j.Data || [];
                        this.appsRetrieved = true;
                        return;
                    })
                    .catch((err) => {
                        console.log("fetch() error: >>", err, "<<");
                        this.msg = "An unknown error occured. Please try again.";
                        this.msgType = msgTypes.danger;
                        return;
                    });

                return;
            },

            //getKeyPairs gets the list of key pairs that have been defined for this 
            //app. The default key pair is automatically chosen so user doesn't have
            //to pick it.
            getKeyPairs: function () {
                let data: Object = {
                    appID: this.appSelectedID,
                    activeOnly: true,
                };
                fetch(get(this.urls.getKeyPairs, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then((j) => {
                        //Check if response is an error from the server.
                        let err: string = handleAPIErrors(j);
                        if (err !== "") {
                            this.msg = err;
                            this.msgType = msgTypes.danger;
                            return;
                        }

                        //Save data to display in GUI.
                        this.keyPairs = j.Data || [];
                        this.keyPairsRetrieved = true;

                        //Set the default keypair in the select menu and the
                        //license data to save.
                        for (let kp of (this.keyPairs as keyPair[])) {
                            if (kp.IsDefault) {
                                this.licenseData.KeyPairID = kp.ID;
                                break;
                            }
                        }

                        return;
                    })
                    .catch((err) => {
                        console.log("fetch() error: >>", err, "<<");
                        this.msg = "An unknown error occured. Please try again.";
                        this.msgType = msgTypes.danger;
                        return;
                    });

                return;
            },

            //getCustomFields gets the list of custom fields that have been defined 
            //for the chosen app.
            getCustomFields: function () {
                let data: Object = {
                    appID: this.appSelectedID,
                    activeOnly: true,
                };
                fetch(get(this.urls.getCustomFields, data))
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

                        //Save data to display in GUI.
                        this.fields = j.Data || [];
                        this.fieldsRetrieved = true;

                        //Set default values.
                        for (let f of (this.fields as customFieldDefined[])) {
                            switch (f.Type) {
                                case customFieldTypeInteger:
                                    f.IntegerValue = f.IntegerDefaultValue;
                                    break;
                                case customFieldTypeDecimal:
                                    f.DecimalValue = f.DecimalDefaultValue;
                                    break;
                                case customFieldTypeText:
                                    f.TextValue = f.TextDefaultValue;
                                    break;
                                case customFieldTypeBoolean:
                                    f.BoolValue = f.BoolDefaultValue;
                                    break;
                                case customFieldTypeMultiChoice:
                                    f.MultiChoiceValue = f.MultiChoiceDefaultValue;
                                    break;
                                case customFieldTypeDate:
                                    f.DateValue = todayPlus(f.DateDefaultIncrement);
                                    break;
                                default:
                                //should never hit this since the field types returned
                                //are only the types we allow to be saved.
                            } //end switch: set default for field type.
                        } //end for: loop through each field setting default value.

                        return;
                    })
                    .catch((err) => {
                        console.log("fetch() error: >>", err, "<<");
                        this.msgLoad = 'An unknown error occured.  Please try again.';
                        this.msgLoadType = msgTypes.danger;
                        return;
                    });

                return;
            },

            //setExpireDate sets the default expiration date for the license to 
            //today's date plus the app's DaysToExpiration value. 
            //
            //This is called when an app is chosen from the select menu.
            setExpireDate: function () {
                for (let a of (this.apps as app[])) {
                    if (a.ID === this.appSelectedID) {
                        this.licenseData.ExpireDate = todayPlus(a.DaysToExpiration);
                        break;
                    }
                }

                return;
            },

            //todayPlusOne calculates the minimum date a license can expire on. This 
            //takes today's date and adds one day to it. 
            //
            //This value is used in the "min" attribute of an input type=date.
            //element.
            todayPlusOne: function () {
                return todayPlus(1);
            },

            //today returns today's date. 
            // 
            //This value is used in the "min" attribute of any custom fields that are
            //of the date type.
            today: function () {
                return getToday();
            },

            //create saves a new license. The data is submitted to the server where a
            //signature is created and the data is saved to the db. Upon success, the
            //user will be redirected to a new page to view the details of this license
            //and download the license file.
            create: function () {
                //Validate.
                this.msgType = msgTypes.danger;
                if (this.appSelectedID < 1) {
                    this.msg = "You must choose an app.";
                    return;
                }
                if (this.licenseData.KeyPairID < 1) {
                    this.msg = "You must choose a key pair.";
                    return;
                }
                if (this.licenseData.CompanyName.trim() === "") {
                    this.msg = "You must provide the company name for which this license is for.";
                    return;
                }
                if (this.licenseData.ContactName.trim() === "") {
                    this.msg = "You must provide the contact name of who requested this license.";
                    return;
                }
                if (this.licenseData.PhoneNumber.trim() === "") {
                    this.msg = "You must provide a phone number.";
                    return;
                }
                if (this.licenseData.Email.trim() === "") {
                    this.msg = "You must provide an email address.";
                    return;
                }
                if (!isEmail(this.licenseData.Email)) {
                    this.msg = "The email address you provided is not valid.";
                    return;
                }
                if (this.licenseData.ExpireDate.trim() === "") {
                    this.msg = "You must provide an expiration date for the license.";
                    return;
                }

                //Custom fields.
                for (let i = 0; i < this.fields.length; i++) {
                    //We don't use a type here since the field as returned from the
                    //API call to build the GUI was a list of "defined" fields, but
                    //now each field has a value so it should be a "result".
                    let cf = this.fields[i];

                    switch (cf.Type) {
                        case customFieldTypeInteger:
                            if (isNaN(cf.IntegerValue) || Number.isNaN(parseInt(cf.IntegerValue.toString())) || !Number.isInteger(cf.IntegerValue)) {
                                this.msg = "You must provide an integer for the " + cf.Name + " field.";
                                return;
                            }
                            if (cf.IntegerValue < cf.NumberMinValue || cf.IntegerValue > cf.NumberMaxValue) {
                                this.msg = "The value for the " + cf.Name + " field must be an integer between " + cf.NumberMinValue.toFixed(0) + " and " + cf.NumberMaxValue.toFixed(0) + ".";
                                return;
                            }
                            break;

                        case customFieldTypeDecimal:
                            if (isNaN(cf.DecimalValue) || Number.isNaN(parseInt(cf.DecimalValue.toString()))) {
                                this.msg = "You must provide an number for the " + cf.Name + " field.";
                                return;
                            }
                            if (cf.DecimalValue < cf.NumberMinValue || cf.DecimalValue > cf.NumberMaxValue) {
                                this.msg = "The value for the " + cf.Name + " field must be an number between " + cf.NumberMinValue.toFixed(2) + " and " + cf.NumberMaxValue.toFixed(2) + ".";
                                return;
                            }
                            break;

                        case customFieldTypeText:
                            //blank values are acceptable for text fields.
                            break;

                        case customFieldTypeBoolean:
                            //bool fields default to false if BoolValue isn't exactly 'true'.
                            break;

                        case customFieldTypeMultiChoice:
                            let ops: string[] = cf.MultiChoiceOptions.split(this.separator);
                            if (!ops.includes(cf.MultiChoiceValue)) {
                                this.msg = "You must choose a value for the " + cf.Name + " field.";
                                return
                            }

                            break;

                        case customFieldTypeDate:
                            if (cf.DateValue.trim() === "") {
                                this.msg = "You must choose a date for the " + cf.Name + "field.";
                                return;
                            }
                            break;

                        default:
                        //This should never happen since list of fields was
                        //retrieved from server/db and we checked types before
                        //data was saved.
                    } //end switch: validate based on field type.

                    //Move ID for field.
                    //
                    //When we looked up the list of fields to build the GUI, we looked
                    //up the defined fields. However, when we submit data to create
                    //a license, we are submitting results. We need to "move" the ID
                    //since submitting the results with and ID wouldn't make sense
                    //since the results haven't been saved to the database yet.
                    //
                    //This is wrapped in a "if" to handle times when form is submitted,
                    //but an error occurs, and user has to resubmit. Without the "if",
                    //the CustomFieldDefinedID will be set to 0 since the ID was set to
                    //0 an a previous failed submission.                    
                    if (cf.CustomFieldDefinedID !== 0 && cf.ID > 0) {
                        cf.CustomFieldDefinedID = cf.ID;
                        cf.ID = 0;
                    }

                    //Unset other defined field data.
                    // 
                    //Sending this data to server is unnecessary since it is for the 
                    //defined field, not for a result.
                    cf.CreatedByUserID = 0;
                    cf.DatetimeCreated = "";
                    cf.DatetimeModified = "";
                    // cf.Instructions =       ""; //Instructions is not unset since doing so causes the GUI to strip the help icon and look incomplete.
                    // cf.Name =               ""; //Name is not unset since doing so causes the GUI to strip the field name and look incomplete.

                } //end for: loop through each field validating each.

                //Make sure data isn't already being submitted.
                if (this.submitting) {
                    console.log("already submitting...");
                    return;
                }

                //Make API request.
                this.msg = "Saving...";
                this.msgType = msgTypes.primary;
                this.submitting = true;

                let data: Object = {
                    licenseData: JSON.stringify(this.licenseData),
                    customFields: JSON.stringify(this.fields),
                };
                fetch(post(this.urls.add, data))
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

                        //Show success message and redirect user to license's page.
                        let licenseID: number = j.Data;
                        if (licenseID === undefined) {
                            this.msg = "Could not determine where to redirect you. This is an odd error...";
                            this.msgType = msgTypes.warning;
                            return;
                        }

                        this.msg = "License created! Redirecting to license...";
                        this.msgType = msgTypes.primary;
                        setTimeout(function () {
                            window.location.href = "/app/licensing/license/?id=" + licenseID;
                        }, defaultTimeout);

                        return;
                    })
                    .catch((err) => {
                        console.log("fetch() error: >>", err, "<<");
                        this.msg = "An unknown error occured. Please try again.";
                        this.msgType = msgTypes.danger;
                        this.submitting = false;
                        return;
                    });

                return;
            },

            //getAPIKeys gets the list of API keys. This is only done if a user clicks
            //the showAPIBuilder button and the list of API keys hasn't already been
            //looked up. We need these to more completely build the API request to 
            //create a license.
            getAPIKeys: function () {
                if (this.apiKeysRetrieved) {
                    return;
                }

                let data: Object = {};
                fetch(get(this.urls.getAPIKeys, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then((j) => {
                        //Check if response is an error from the server.
                        let err: string = handleAPIErrors(j);
                        if (err !== "") {
                            this.apiBuilderMsg = err;
                            this.apiBuilderMsgType = msgTypes.danger;
                            return;
                        }

                        this.apiKeys = j.Data || [];
                        this.apiKeysRetrieved = true;
                        return;
                    })
                    .catch((err) => {
                        console.log("fetch() error: >>", err, "<<");
                        this.apiBuilderMsg = "An unknown error occured. Please try again.";
                        this.apiBuilderMsgType = msgTypes.danger;
                        return;
                    });

                return;
            },

            //buildAPIExample builds the curl request to create an API key from the
            //data the user chosen/provided. This is done to show the user an example
            //of an API call to create a license.
            //
            //We have to handle email specially due to @ sign. curl uses @ sign to
            //submit a file.
            buildAPIExample: function () {
                //Validate.
                if (this.appSelectedID < 1 || this.licenseData.KeyPairID < 1 || !this.fieldsRetrieved) {
                    return;
                }

                //Gather info to build host part of example. Just reuse the host that
                //the user currently sees in the browser since that should be right.
                let proto: string = window.location.protocol;
                let host: string = window.location.host;

                //Handle custom fields by arranging them as an object of key:value
                //pairs. 
                let cf: { [key: string]: any } = {};
                for (let f of this.fields) {
                    switch (f.Type) {
                        case customFieldTypeInteger:
                            cf[f.Name] = f.IntegerValue;
                            break;
                        case customFieldTypeDecimal:
                            cf[f.Name] = f.DecimalValue;
                            break;
                        case customFieldTypeText:
                            cf[f.Name] = f.TextValue;
                            break;
                        case customFieldTypeBoolean:
                            cf[f.Name] = f.BoolValue;
                            break;
                        case customFieldTypeMultiChoice:
                            cf[f.Name] = f.MultiChoiceValue;
                            break;
                        case customFieldTypeDate:
                            cf[f.Name] = f.DateValue;
                            break;
                        default:
                        //don't do anything here, this should never occur since
                        //we looked up custom fields from the db.
                    }
                } //end for: loop through each custom field defined.

                let cfEncoded = encodeURIComponent(JSON.stringify(cf));

                //Build the lines of the example.
                let lines: string[] = [
                    "curl '" + proto + "//" + host + "/api/v1/licenses/add/'",
                    "-X POST",
                    "-H 'Content-type: application/x-www-form-urlencoded'",
                    "-H 'Authorization: Bearer " + this.apiKeySelected + "'",
                    "-d appID='" + this.appSelectedID + "'",
                    "-d keyPairID='" + this.licenseData.KeyPairID + "'",
                    "-d companyName='" + this.licenseData.CompanyName + "'",
                    "-d contactName='" + this.licenseData.ContactName + "'",
                    "-d phoneNumber='" + this.licenseData.PhoneNumber + "'",
                    "-d email='" + this.licenseData.Email + "'",
                    "-d expireDate='" + this.licenseData.ExpireDate + "'",
                    "-d fields=" + cfEncoded + "",
                ];

                if (this.returnLicenseFile) {
                    lines.push("-d returnLicenseFile=true");
                }

                let s: string = lines.join(" \\\n");

                this.apiExample = s;
                return
            },
        },

        mounted() {
            //Load the apps the user can choose from.
            this.getApps();
            return;
        }
    }).mount("#createLicense");
}