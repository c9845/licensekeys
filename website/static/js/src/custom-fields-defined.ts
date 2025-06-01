/**
 * customFieldsDefined.ts
 * 
 * This file handles adding, viewing, editing, and deleting the custom fields defined
 * for an app. Custom fields allow you to add arbitrary data to your license files for
 * use such as limiting user count, encoding when support expired, enabling/disabling
 * features, etc.
 */

import { createApp } from "vue";
import { msgTypes, apiBaseURL, defaultTimeout } from "./common";
import { get, post, handleRequestErrors, getJSON, handleAPIErrors } from "./fetch";
import { customFieldDefined, customFieldTypes, customFieldTypeInteger, customFieldTypeDecimal, customFieldTypeText, customFieldTypeBoolean, customFieldTypeMultiChoice, customFieldTypeDate } from "./types.ts";

//Manage the list of custom fields defined.
export var listCustomFieldsDefined: any; //must be "any", not "ComponentPublicInstance" to remove errors when calling functions (methods) of this Vue instance.
if (document.getElementById("listCustomFieldsDefined")) {
    listCustomFieldsDefined = createApp({
        name: 'listCustomFieldsDefined',

        compilerOptions: {
            delimiters: ['[[', ']]'],
        },

        data() {
            return {
                //App to look up key pairs for. This is populated by setAppID.
                appSelectedID: 0,

                //List of custom fields.
                fields: [] as customFieldDefined[],
                fieldsRetrieved: false,
                msgLoad: "",
                msgLoadType: "",

                //Handle GUI state. Collapse the card to take up less screen space.
                collapseUI: false,

                //Need these for v-if in html.
                customFieldTypeInteger: customFieldTypeInteger,
                customFieldTypeDecimal: customFieldTypeDecimal,
                customFieldTypeText: customFieldTypeText,
                customFieldTypeBoolean: customFieldTypeBoolean,
                customFieldTypeMultiChoice: customFieldTypeMultiChoice,
                customFieldTypeDate: customFieldTypeDate,

                //Endpoints.
                urls: {
                    get: apiBaseURL + "custom-fields/defined/",
                }
            }
        },

        methods: {
            //setAppID sets the ID of the chosen app in this Vue object. 
            //
            //This is called from manageApps.setAppIDInOtherVueObjects() when an app 
            //is chosen from the list of defined apps. This then retrieves the list 
            //of key pairs for this app.
            setAppID: function (appID: number) {
                //Save the selected app's ID.
                this.appSelectedID = appID;

                //Handle adding a new app. No need to make API call when GUI is set
                //for adding a new app.
                if (appID === 0) {
                    this.keyPairs = [];
                    this.msgLoad = "";
                    return;
                }

                //Get the list of keypairs for the selected app.
                this.getCustomFields();
                return;
            },

            //getCustomFields gets the list of custom fields that have been defined 
            //for this app.
            getCustomFields: function () {
                let data: Object = {
                    appID: this.appSelectedID,
                    activeOnly: true,
                };
                fetch(get(this.urls.get, data))
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

                        this.fields = j.Data || [];
                        this.fieldsRetrieved = true;
                        return;
                    })
                    .catch((err) => {
                        console.log("fetch() error: >>", err, "<<");
                        this.msgLoad = "An unknown error occured. Please try again.";
                        this.msgLoadType = msgTypes.danger;
                        return;
                    });

                return;
            },

            //passToManageCustomFieldModal passes data about a chosen custom field to 
            //the modal so we can display more info about the custom field than we 
            //can in a table. This is done so that we don't have to pass along just 
            //the customFieldID and the perform a GET request to look up the field's 
            //data.
            //
            //This is called when clicking the "settings" button for a custom field or
            //clicking the "add" button to add a new custom field.
            //
            //When adding, "undefined" is passed as the custom field.
            //
            //Note that this does not open the modal, that is handled through bootstrap
            //data-toggle and data-target attributes.
            passToManageCustomFieldModal: function (cfd: customFieldDefined | undefined) {
                modalManageCustomFieldDefined.setCustomFieldInModal(this.appSelectedID, cfd);
                return
            },
        }
    }).mount("#listCustomFieldsDefined");
}

//Handle displaying the full details of a custom field defined, and allow management
//of it, or add a new custom field.
var modalManageCustomFieldDefined: any; //must be "any", not "ComponentPublicInstance" to remove errors when calling functions (methods) of this Vue instance.
if (document.getElementById("modal-manageCustomFieldDefined")) {
    modalManageCustomFieldDefined = createApp({
        name: 'modalManageCustomFieldDefined',

        compilerOptions: {
            delimiters: ['[[', ']]'],
        },

        data() {
            return {
                //App the custom field is for. We mostly need this for adding a custom
                //field.
                // 
                //This is set in setCustomFieldInModal().
                appSelectedID: 0,

                //The custom field's data. Either for an existing custom field or for 
                //a new custom field a user is adding.
                //
                //This is set in setCustomFieldInModal();
                fieldData: {} as customFieldDefined,

                //Options to choose from when adding.
                customFieldTypes: customFieldTypes,

                //Need these for v-if in html.
                customFieldTypeInteger: customFieldTypeInteger,
                customFieldTypeDecimal: customFieldTypeDecimal,
                customFieldTypeText: customFieldTypeText,
                customFieldTypeBoolean: customFieldTypeBoolean,
                customFieldTypeMultiChoice: customFieldTypeMultiChoice,
                customFieldTypeDate: customFieldTypeDate,

                //Separator for multichoice options.
                separator: ";",

                //Form submissing stuff.
                submitting: false,
                msgSave: '',
                msgSaveType: '',

                //Endpoints.
                urls: {
                    add: apiBaseURL + "custom-fields/defined/add/",
                    update: apiBaseURL + "custom-fields/defined/update/",
                    delete: apiBaseURL + "custom-fields/defined/delete/",
                }
            }
        },
        computed: {
            //adding is set to true when the user is adding a custom field.
            // 
            //This is used to modify what the GUI displays (modal title and body text) 
            //to show correct helpful information based on adding or editing a field.
            adding: function () {
                if (this.fieldData.ID === undefined || this.fieldData.ID < 1) {
                    return true;
                }

                return false;
            },

            //multiChoiceOptions returns the user-provided semicolon separated options 
            //as an array for building the select menu where a user can choose the 
            //default option.
            multiChoiceOptions: function () {
                //Page is just loading, mounted hasn't been called yet to set default 
                //empty state.
                if (this.fieldData.MultiChoiceOptions === undefined) {
                    return [];
                }

                //Ignore when field isn't a multi choice.
                if (this.fieldData.Type !== this.customFieldTypeMultiChoice) {
                    return [];
                }

                //No options provided by user yet.
                if (this.fieldData.MultiChoiceOptions.trim().length === 0) {
                    return [];
                }

                //Options provided, split string into array for use with v-for in HTML.
                let opsArray: string[] = this.fieldData.MultiChoiceOptions.split(this.separator);
                let validatedArray: string[] = [];
                for (let s of opsArray) {
                    s = s.trim();

                    //Ignore blanks.
                    if (s === "") {
                        continue;
                    }

                    //Ignore duplicates.
                    if (validatedArray.includes(s)) {
                        continue;
                    }

                    validatedArray.push(s);
                }

                return validatedArray;
            }
        },

        methods: {
            //setCustomFieldInModal is used to populate the modal with data about a
            //custom fields, or to set the modal to a clean state for adding a new
            //custom field. 
            //
            //This is called from listKeyPairs.passToManageKeypairModal() upon a user
            //clicking the "edit" or "add" buttons.
            setCustomFieldInModal: function (appID: number, cfd: customFieldDefined | undefined) {
                //Always save the app ID.
                this.appSelectedID = appID;

                //Always reset.
                this.resetForm();

                //User wants to add a new custom field.
                if (cfd === undefined) {
                    return;
                }

                //User is viewing details of a custom field.
                this.fieldData = cfd;
                return;
            },

            //resetForm sets the modal back to a clean state for adding a new custom 
            //field.
            resetForm: function () {
                this.fieldData = {
                    ID: 0,
                    DatetimeCreated: "", //wont be set, just to match type.
                    DatetimeModified: "", //" " "
                    CreatedByUserID: 0, //" " "
                    Active: true,
                    AppID: this.appSelectedID,
                    Name: "",
                    Type: "",
                    Instructions: "",
                    IntegerDefaultValue: 0,
                    DecimalDefaultValue: 0,
                    TextDefaultValue: "",
                    BoolDefaultValue: false,
                    MultiChoiceDefaultValue: "",
                    DateDefaultIncrement: 0,
                    NumberMinValue: 0,
                    NumberMaxValue: 0,
                    MultiChoiceOptions: "",
                } as customFieldDefined;

                this.submitting = false;
                this.msgSave = "";
                this.msgSaveType = "";
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

                if (this.fieldData.Name.trim() === "") {
                    this.msgSave = "You must provide a name for this field.";
                    return;
                }

                switch (this.fieldData.Type) {
                    case customFieldTypeInteger:
                        if (this.fieldData.NumberMinValue === undefined || this.fieldData.NumberMaxValue === undefined) {
                            this.msgSave = "You must provide both the minimum and maximum value for the numbers you allow in this field.";
                            return;
                        }
                        if (!Number.isInteger(this.fieldData.NumberMinValue) || !Number.isInteger(this.fieldData.NumberMinValue)) {
                            this.msgSave = "The minimum and maximum values must be integers (no decimals).";
                            return;
                        }
                        if (this.fieldData.NumberMinValue >= this.fieldData.NumberMaxValue) {
                            this.msgSave = "The minimum value must be less than the maximum.";
                            return;
                        }
                        if (this.fieldData.IntegerDefaultValue === undefined) {
                            this.msgSave = "You must provide a default value for this field.";
                            return;
                        }
                        if (!Number.isInteger(this.fieldData.IntegerDefaultValue)) {
                            this.msgSave = "The default must be an integer (no decimals).";
                            return;
                        }
                        if (this.fieldData.IntegerDefaultValue < this.fieldData.NumberMinValue || this.fieldData.IntegerDefaultValue > this.fieldData.NumberMaxValue) {
                            this.msgSave = "The default value must be within the minimum to maximum range.";
                            return;
                        }
                        break;

                    case customFieldTypeDecimal:
                        if (this.fieldData.NumberMinValue === undefined || this.fieldData.NumberMaxValue === undefined) {
                            this.msgSave = "You must provide both the minimum and maximum value for the numbers you allow in this field.";
                            return;
                        }
                        if (this.fieldData.NumberMinValue >= this.fieldData.NumberMaxValue) {
                            this.msgSave = "The minimum value must be less than the maximum.";
                            return;
                        }
                        if (this.fieldData.DecimalDefaultValue === undefined) {
                            this.msgSave = "You must provide a default value for this field.";
                            return;
                        }
                        if (this.fieldData.DecimalDefaultValue < this.fieldData.NumberMinValue || this.fieldData.DecimalDefaultValue > this.fieldData.NumberMaxValue) {
                            this.msgSave = "The default value must be within the minimum to maximum range.";
                            return;
                        }
                        break;

                    case customFieldTypeText:
                        if (this.fieldData.TextDefaultValue === undefined || this.fieldData.TextDefaultValue.trim() === "") {
                            this.msgSave = "You must provide a default value for this field.";
                            return;
                        }
                        break;

                    case customFieldTypeBoolean:
                        if (this.fieldData.BoolDefaultValue === undefined || (this.fieldData.BoolDefaultValue !== true && this.fieldData.BoolDefaultValue !== false)) {
                            this.msgSave = "You must choose a default value for this field.";
                            return;
                        }
                        break;

                    case customFieldTypeMultiChoice:
                        if (this.fieldData.MultiChoiceOptions === undefined || this.fieldData.MultiChoiceOptions === "") {
                            this.msgSave = "You must provide at least one option.";
                            return;
                        }

                        //When validating multi choice options, we handle the provided 
                        //options as an array. This allows for easier sanitizing and 
                        //validating since we are going to display the options in a 
                        //select from an array anyway.
                        let opsArray: string[] = this.fieldData.MultiChoiceOptions.split(this.separator);

                        //Remove starting/ending whitespace from each option. Only 
                        //put non-blank results into output array. Make sure element 
                        //doesn't already exist in output array, warning if duplicate 
                        //is found.
                        let validatedArray: string[] = [];
                        for (let s of opsArray) {
                            s = s.trim();

                            if (s === "") {
                                continue;
                            }

                            if (validatedArray.includes(s)) {
                                this.msgSave = 'The option "' + s + '" is included more than once. Please remove the duplicate(s).';
                                return;
                            }

                            validatedArray.push(s);
                        }

                        //Make sure something exists in output array. This catch if 
                        //user didn't provide any values or a blank value only.
                        if (validatedArray.length === 0) {
                            this.msgSave = "You must provide as least one option for this field.";
                            return;
                        }

                        //Make sure default value is in list of options.
                        if (this.fieldData.MultiChoiceDefaultValue === undefined || this.fieldData.MultiChoiceDefaultValue.trim() === "") {
                            this.msgSave = "You must choose a default value for this field.";
                            return;
                        }
                        if (!validatedArray.includes(this.fieldData.MultiChoiceDefaultValue)) {
                            this.msgSave = "Please choose a default value from the list of options your provided.";
                            return;
                        }

                        //Set validated array back.
                        this.fieldData.MultiChoiceOptions = validatedArray.join(this.separator);
                        break;

                    case customFieldTypeDate:
                        if (this.fieldData.DateDefaultIncrement === undefined || this.fieldData.DateDefaultIncrement < 0) {
                            this.msgSave = "Please provide the number of days from the date the license is created to calculate the date set for this field.";
                            return;
                        }
                        break;

                    default:
                        this.msgSave = "Please choose an field type from the provided options.";
                        return;
                }

                //Perform correct task.
                if (this.fieldData.ID !== undefined && this.fieldData.ID > 0) {
                    this.update();
                }
                else {
                    this.add();
                }

                return;
            },

            //add saves a new field.
            add: function () {
                //Validation.

                //Make API request.
                this.msgSave = "Adding...";
                this.msgSaveType = msgTypes.primary;
                this.submitting = true;

                let data: Object = {
                    data: JSON.stringify(this.fieldData),
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

                        //Refresh the list of fields.
                        listCustomFieldsDefined.getCustomFields();

                        //Show success and reset the form.
                        this.msgSave = "Field added!";
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
                        this.msgSave = "An unknown error occured. Please try again.";
                        this.msgSaveType = msgTypes.danger;
                        this.submitting = false;
                        return;
                    });

                return;
            },

            //update saves changes to an existing field.
            update: function () {
                //Validation.
                if (isNaN(this.fieldData.ID) || this.fieldData.ID === '' || this.fieldData.ID < 1) {
                    this.msgSave = "Could not determine which field you are trying to update. Please refresh the page and try again.";
                    this.msgSaveType = msgTypes.danger;
                    return;
                }

                //Make API request.
                this.msgSave = "Saving...";
                this.msgSaveType = msgTypes.primary;
                this.submitting = true;

                let data: Object = {
                    data: JSON.stringify(this.fieldData),
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
                        this.msgSave = "An unknown error occured. Please try again.";
                        this.msgSaveType = msgTypes.danger;
                        this.submitting = false;
                        return;
                    });

                return;
            },

            //remove marks a field as inactive. Inactive field will no long show up 
            //in the list of fields for this app or be required when creating a new 
            //license.
            remove: function () {
                //Make sure data isn't already being submitted.
                if (this.submitting) {
                    console.log("already submitting...");
                    return;
                }

                //Make sure we know what field we are deleting.
                if (isNaN(this.fieldData.ID) || this.fieldData.ID === '' || this.fieldData.ID < 1) {
                    this.msgSave = "Could not determine which field you are trying to delete. Please refresh the page and try again.";
                    this.msgSaveType = msgTypes.danger;
                    return;
                }

                //Make API request.
                this.msgSave = "Deleting...";
                this.msgSaveType = msgTypes.primary;
                this.submitting = true;

                let data: Object = {
                    id: this.fieldData.ID,
                };
                fetch(post(this.urls.delete, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then((j) => {
                        //Check if response is an error from the server.
                        let err: string = handleAPIErrors(j);
                        if (err !== "") {
                            this.msgSave = err;
                            this.msgSaveType = msgTypes.danger;
                            return;
                        }

                        //Refresh the list of custom fields in table. The modal will
                        //be closed by the data-dismiss on the button clicked that
                        //called this function.
                        listCustomFieldsDefined.getCustomFields();
                        this.submitting = false;

                        return;
                    })
                    .catch((err) => {
                        console.log("fetch() error: >>", err, "<<");
                        this.msgSave = "An unknown error occured. Please try again.";
                        this.msgSaveType = msgTypes.danger;
                        this.submitting = false;
                        return;
                    });
                return;
            },
        }
    }).mount("#modal-manageCustomFieldDefined")
}