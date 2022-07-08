/**
 * create-license.ts
 * This file handles creating a new license. The user picks an app and a keypair,
 * then provides the common fields and custom fields, if needed, and the license
 * is created and signed using the keypair. The user is directed to a page for
 * managing the license and downloading the license file.
 */

/// <reference path="common.ts" />
/// <reference path="fetch.ts" />
/// <reference path="types.ts" />

if (document.getElementById("createLicense")) {
    //@ts-ignore cannot find name Vue
    var createLicense = new Vue({
        name: 'createLicense',
        delimiters: ['[[', ']]'],
        el: '#createLicense',
        data: {
            //loading apps from api call
            apps:           [] as app[], 
            appsRetrieved:  false, //set to true once api call is complete, whether or not any apps were found. used to show loading message in select.
            appSelectedID:  0,
            
            //loading keypairs from api call, based on app chosen.
            //default keypair is automatically set for license.
            keyPairs:           [] as keyPair[],
            keyPairsRetrieved:  false,

            //loading any custom fields, based on app chosen
            //This will also store the value chosen for each field. We do this
            //since it would just be messy to store the field ID and value in
            //another array of objects. While this doesn't end up being a "defined"
            //field, a "defined" and "result" field are similar enough to do this.
            //We translate a defined field to a result when saving the license data.
            fields:             [],
            fieldsRetrieved:    false,

            //need these for v-if in html
            customFieldTypeInteger:     customFieldTypeInteger,
            customFieldTypeDecimal:     customFieldTypeDecimal,
            customFieldTypeText:        customFieldTypeText,
            customFieldTypeBoolean:     customFieldTypeBoolean,
            customFieldTypeMultiChoice: customFieldTypeMultiChoice,
            customFieldTypeDate:        customFieldTypeDate,

            separator: ";", //separator for multichoice options

            //data for license being created
            licenseData: {
                KeyPairID:          0, //from chosen/default keypair for app
                CompanyName:        "ACME Dynamite",
                ContactName:        "Wyle E Coyote",
                PhoneNumber:        "123-555-1212",
                Email:              "wyle@example.com",
                ExpireDate:         "", //by default, this is set to "today" plus the app's DaysToExpiration
            } as license,

            //errors when loading data or creating license
            submitting: false,
            msg:        '',
            msgType:    '',

            //endpoints
            urls: {
                getApps:            "/api/apps/",
                getKeyPairs:        "/api/key-pairs/",
                getCustomFields:    "/api/custom-fields/defined/",
                add:                "/api/licenses/add/",
                getAPIKeys:         "/api/api-keys/",
            },

            //Used for displaying the API builder. This shows a GUI of the API
            //request a user would need to build to create a license via the API.
            showAPIBuilder:     false,
            returnLicenseFile:  false,
            apiKeys:            [] as apiKey[],
            apiKeysRetrieved:   false,
            apiBuilderMsg:      "",
            apiBuilderMsgType:  "",
            apiKeySelected:     "", //not the ID, but the actual key
        },
        methods: {
            //setField saves a chosen radio toggle's value to the Vue object. We
            //use index here, not name, since we know the index of the fields
            //for this app when we built the GUI.
            setField: function (idx: number, value: boolean) {
                this.fields[idx].BoolValue = value;
                return;
            },

            //getApps gets the list of apps that have been defined.
            getApps: function () {
                let data: Object = {};
                fetch(get(this.urls.getApps, data))
                .then(handleRequestErrors)
                .then(getJSON)
                .then(function (j) {
                    //check if response is an error from the server
                    let err: string = handleAPIErrors(j);
                    if (err !== '') {
                        createLicense.msg =     err;
                        createLicense.msgType = msgTypes.danger;
                        return;
                    }
    
                    //save data to display in gui
                    createLicense.apps =           j.Data || [];
                    createLicense.appsRetrieved =  true;
    
                    return;
                })
                .catch(function (err) {
                    console.log("fetch() error: >>", err, "<<");
                    createLicense.msg =     'An unknown error occured. Please try again.';
                    createLicense.msgType = msgTypes.danger;
                    return;
                });
    
                return;
            },

            //getKeyPairs gets the list of key pairs that have been defined for this 
            //app. The default key pair is automatically chosen so user doesn't have
            //to pick it.
            getKeyPairs: function () {
                let data: Object = {
                    appID:      this.appSelectedID,
                    activeOnly: true,
                };
                fetch(get(this.urls.getKeyPairs, data))
                .then(handleRequestErrors)
                .then(getJSON)
                .then(function (j) {
                    //check if response is an error from the server
                    let err: string = handleAPIErrors(j);
                    if (err !== '') {
                        createLicense.msg =     err;
                        createLicense.msgType = msgTypes.danger;
                        return;
                    }
    
                    //save data to display in gui
                    createLicense.keyPairs =             j.Data || [];
                    createLicense.keyPairsRetrieved =    true;

                    //set the default keypair in the select menu and the
                    //license data to save
                    for (let kp of (createLicense.keyPairs as keyPair[])) {
                        if (kp.IsDefault) {
                            createLicense.licenseData.KeyPairID = kp.ID;
                            break;
                        }
                    }
    
                    return;
                })
                .catch(function (err) {
                    console.log("fetch() error: >>", err, "<<");
                    createLicense.msg =     'An unknown error occured. Please try again.';
                    createLicense.msgType = msgTypes.danger;
                    return;
                });
    
                return;
            },

            //getCustomFields gets the list of custom fields that have been defined 
            //for the chosen app.
            getCustomFields: function () {
                let data: Object = {
                    appID:      this.appSelectedID,
                    activeOnly: true,
                };
                fetch(get(this.urls.getCustomFields, data))
                .then(handleRequestErrors)
                .then(getJSON)
                .then(function (j) {
                    //check if response is an error from the server
                    let err: string = handleAPIErrors(j);
                    if (err !== '') {
                        createLicense.msgLoad =     err;
                        createLicense.msgLoadType = msgTypes.danger;
                        return;
                    }
    
                    //save data to display in gui
                    createLicense.fields =            j.Data || [];
                    createLicense.fieldsRetrieved =   true;

                    //set default values
                    for (let f of (createLicense.fields as customFieldDefined[])) {
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
                                //@ts-ignore cannot find Vue
                                Vue.nextTick(function() {
                                    //add cf_bool_id_ to id value to give it some context and not be just a number.
                                    let elemID: string = 'cf_bool_id_' + f.ID.toString();
                                    setToggle(elemID, f.BoolDefaultValue);
                                });
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
                .catch(function (err) {
                    console.log("fetch() error: >>", err, "<<");
                    createLicense.msgLoad =     'An unknown error occured.  Please try again.';
                    createLicense.msgLoadType = msgTypes.danger;
                    return;
                });
    
                return;
            },

            //setExpireDate sets the default expiration date for the license to 
            //today's date plus the app's DaysToExpiration value. This is called when 
            //an app is chosen from the select menu.
            setExpireDate: function() {
                for (let a of (this.apps as app[])) {
                    if (a.ID === this.appSelectedID) {
                        this.licenseData.ExpireDate = todayPlus(a.DaysToExpiration);
                        break;
                    }
                }
                
                return;
            },

            //todayPlusOne calculates the minimum date a license can expire on. This 
            //takes today's date and adds one day to it. This returns a yyyy-mm-dd 
            //formatted string for use in the min attribute on an input type=date 
            //element.
            todayPlusOne: function() {
                return todayPlus(1);
            },

            //today returns today's data and is used as the minimum value in date 
            //inputs. This returns a yyyy-mm-dd formatted string for use in the min 
            //attribute on an input type=date element. 
            today: function() {
                let now: Date = new Date();
                let y: number = now.getFullYear();
                let m: number = now.getMonth() + 1;
                let d: number = now.getDate();
                let yy: string = y.toString();
                let mm: string = (m < 10) ? "0" + m.toString() : m.toString();
                let dd: string = (d < 10) ? "0" + d.toString() : d.toString();
            
                return yy + "-" + mm + "-" + dd;
            },

            //create saves a new license. The data is submitted to the server where a
            //signature is created and the data is saved to the db. Upon success, the
            //user will be redirected to a new page to view the details of this license
            //and download the license file.
            create: function() {
                //validate
                //common fields
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

                //custom fields
                for (let i = 0; i < this.fields.length; i++) {
                    //We don't use a type here since the field as returned from the
                    //API call to build the GUI was a list of "defined" fields, but
                    //now each field has a value so it should be a "result".
                    let cf = this.fields[i];

                    switch (cf.Type) {
                        case customFieldTypeInteger:
                            if (isNaN(cf.IntegerValue) || parseInt(cf.IntegerValue.toString()) === NaN || !Number.isInteger(cf.IntegerValue)) {
                                this.msg = "You must provide an integer for the " + cf.Name + " field.";
                                return;
                            }
                            if (cf.IntegerValue < cf.NumberMinValue || cf.IntegerValue > cf.NumberMaxValue) {
                                this.msg = "The value for the " + cf.Name + " field must be an integer between " + cf.NumberMinValue.toFixed(0) + " and " + cf.NumberMaxValue.toFixed(0) + ".";
                                return;
                            }
                            break;

                        case customFieldTypeDecimal:
                            if (isNaN(cf.DecimalValue) || parseInt(cf.DecimalValue.toString()) === NaN) {
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

                    //Unset other defined field data. Sending this data to server is
                    //unnecessary since it is for the defined field, not for a result.
                    cf.CreatedByUserID =    0;
                    cf.DatetimeCreated =    "";
                    cf.DatetimeModified =   "";
                    // cf.Instructions =       ""; //Instructions is not unset since doing so causes the GUI to strip the help icon and look incomplete.
                    // cf.Name =               ""; //Name is not unset since doing so causes the GUI to strip the field name and look incomplete.

                } //end for: loop through each field validating each.
                
                //make sure data isn't already being submitted
                if (this.submitting) {
                    console.log("already submitting...");                
                    return;
                }

                //validation ok
                this.msg =          "Saving...";
                this.msgType =      msgTypes.primary;
                this.submitting =   true;

                //perform api call
                let data: Object = {
                    licenseData:    JSON.stringify(this.licenseData),
                    customFields:   JSON.stringify(this.fields),
                };
                fetch(post(this.urls.add, data))
                .then(handleRequestErrors)
                .then(getJSON)
                .then(function (j) {
                    //check if response is an error from the server
                    let err: string = handleAPIErrors(j);
                    if (err !== '') {
                        createLicense.msg =         err;
                        createLicense.msgType =     msgTypes.danger;
                        createLicense.submitting =  false;
                        return;
                    }

                    //show success message and redirect user to license's page
                    let licenseID: number = j.Data;
                    if (licenseID === undefined) {
                        createLicense.msg =     "Could not determine where to redirect you. This is an odd error...";
                        createLicense.msgType = msgTypes.warning;
                        return;
                    }
                    
                    createLicense.msg =     "License created! Redirecting to license...";
                    createLicense.msgType = msgTypes.primary;
                    setTimeout(function() {
                        window.location.href = "/license/?id=" + licenseID;
                    }, defaultTimeout);

                    return;
                })
                .catch(function (err) {
                    console.log("fetch() error: >>", err, "<<");
                    createLicense.msg =         'An unknown error occured. Please try again.';
                    createLicense.msgType =     msgTypes.danger;
                    createLicense.submitting =  false;
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
                .then(function (j) {
                    //check if response is an error from the server
                    let err: string = handleAPIErrors(j);
                    if (err !== '') {
                        createLicense.apiBuilderMsg =       err;
                        createLicense.apiBuilderMsgType =   msgTypes.danger;
                        return;
                    }
    
                    createLicense.apiKeys =            j.Data || [];
                    createLicense.apiKeysRetrieved =   true;
                    return;
                })
                .catch(function (err) {
                    console.log("fetch() error: >>", err, "<<");
                    createLicense.apiBuilderMsg =       'An unknown error occured.  Please try again.';
                    createLicense.apiBuilderMsgType =   msgTypes.danger;
                    return;
                });
    
                return;
            },
        },
        computed: {
            //getAPIRequestFieldsObject builds the object that stores the custom
            //fields defined for an API request.
            getAPIRequestFieldsObject: function() {
                let cf: object = {};
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
                } //end for

                return cf;
            },

            //showAPIRequest builds the curl request to create an API key from the
            //data the user chosen/provided. This is done to show the user an example
            //of an API call to create a license.
            //
            //We have to handle email specially due to @ sign. curl uses @ sign to
            //submit a file.
            showAPIRequest: function() {
                if (this.appSelectedID < 1 || this.licenseData.KeyPairID < 1 || !this.fieldsRetrieved) {
                    return;
                }

                let proto: string = window.location.protocol;
                let host: string =  window.location.host;

                let cf = encodeURIComponent(JSON.stringify(this.getAPIRequestFieldsObject));

                let lines: string[] = [
                    "curl '" + proto + "//" + host + "/api/v1/licenses/add/'",
                    "-X POST",
                    "-H 'Content-type: application/x-www-form-urlencoded'",
                    "-d apiKey='" + this.apiKeySelected + "'",
                    "-d appID='" + this.appSelectedID + "'",
                    "-d keyPairID='" + this.licenseData.KeyPairID + "'",
                    "-d companyName='" + this.licenseData.CompanyName + "'",
                    "-d contactName='" + this.licenseData.ContactName + "'",
                    "-d phoneNumber='" + this.licenseData.PhoneNumber + "'",
                    "-d email='" + this.licenseData.Email + "'",
                    "-d expireDate='" + this.licenseData.ExpireDate + "'",
                    "-d fields=" + cf + "",
                ];

                if (this.returnLicenseFile) {
                    lines.push("-d returnLicenseFile=true");
                }

                let s: string = lines.join(" \\\n");
                return s;
            },
        },
        mounted() {
            //Load the apps the user can choose from.
            this.getApps();

            //Set some default stuff.
            setToggle("returnLicenseFile", false);

            return;
        }
    })
}