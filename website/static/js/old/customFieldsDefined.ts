/**
 * customFieldsDefined.ts
 * This file handles adding, viewing, editing, and deleting the custom fields defined
 * for an app. Custom fields allow you to add arbitrary data to your license files for
 * use such as limiting user count, encoding when support expired, enabling/disabling
 * features, etc.
 */

/// <reference path="common.ts" />
/// <reference path="fetch.ts" />
/// <reference path="types.ts" />

if (document.getElementById("listCustomFieldsDefined")) {
    //listCustomFieldsDefined handles displaying the list of custom fields for an app. 
    //This does not handle adding or updating custom fields which is done in a modal.
    //@ts-ignore cannot find name Vue
    var listCustomFieldsDefined = new Vue({
        name: 'listCustomFieldsDefined',
        delimiters: ['[[', ']]'],
        el: '#listCustomFieldsDefined',
        data: {
            //App to look up custom fields for. This is populated by setAppID.
            appSelectedID: 0,

            //List of custom fields for this app.
            fields: [] as customFieldDefined[],
            fieldsRetrieved: false,

            //need these for v-if in html
            customFieldTypeInteger: customFieldTypeInteger,
            customFieldTypeDecimal: customFieldTypeDecimal,
            customFieldTypeText: customFieldTypeText,
            customFieldTypeBoolean: customFieldTypeBoolean,
            customFieldTypeMultiChoice: customFieldTypeMultiChoice,
            customFieldTypeDate: customFieldTypeDate,

            //errors
            msgLoad: '',
            msgLoadType: '',

            collapseUI: false, //collapse the card to take up less screen space.

            //endpoints
            urls: {
                get: "/api/custom-fields/defined/",
            }
        },
        methods: {
            //setAppID sets the appSelectedID value in this vue object. This is called from
            //manageApps.setAppInOtherVueObjects() when an app is chosen from the list of
            //defined apps. This then retrieves the list of custom fields for this app.
            setAppID: function (appID: number) {
                this.appSelectedID = appID;

                //handle adding a new app.
                if (appID === 0) {
                    this.fields = [];
                    this.msgLoad = "";
                    return;
                }

                this.getCustomFields();
                return;
            },

            //getCustomFields gets the list of custom fields that have been defined for this 
            //app.
            getCustomFields: function () {
                let data: Object = {
                    appID: this.appSelectedID,
                    activeOnly: true,
                };
                fetch(get(this.urls.get, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then(function (j) {
                        //check if response is an error from the server
                        let err: string = handleAPIErrors(j);
                        if (err !== '') {
                            listCustomFieldsDefined.msgLoad = err;
                            listCustomFieldsDefined.msgLoadType = msgTypes.danger;
                            return;
                        }

                        //save data to display in gui
                        listCustomFieldsDefined.fields = j.Data || [];
                        listCustomFieldsDefined.fieldsRetrieved = true;

                        return;
                    })
                    .catch(function (err) {
                        console.log("fetch() error: >>", err, "<<");
                        listCustomFieldsDefined.msgLoad = 'An unknown error occured. Please try again.';
                        listCustomFieldsDefined.msgLoadType = msgTypes.danger;
                        return;
                    });

                return;
            },

            //passToModal handles the clicking of buttons/icons that open the add/edit custom
            //field modal. When adding a new custom field, 'undefined' is simply passed along. 
            //But when editing the details of a custom field (full data was already retrieved 
            //to show list of custom fields), this passes along an object representing the custom
            //field's data. This is done so that we don't need to retrieve the full data about a 
            //custom field again and we can reuse the same modal for adding or editting.
            //
            //Note that this does not open the modal. Opening of the modal is handled through
            //bootstrap data-toggle and data-target attributes.
            passToModal: function (item: keyPair | undefined) {
                modalCustomFieldDefined.setModalData(item);
                return
            },
        }
    })
}

if (document.getElementById("modal-customFieldDefined")) {
    //modalCustomFieldDefined handles displaying the full details of a custom field, adding 
    //a new custom field, and editing a custom field.
    //@ts-ignore cannot find name Vue
    var modalCustomFieldDefined = new Vue({
        name: 'modalCustomFieldDefined',
        delimiters: ['[[', ']]'],
        el: '#modal-customFieldDefined',
        data: {
            //App the field is for. This is populated by setAppID. This is mostly used for
            //setting the modal form to a default state for adding a new custom field.
            appSelectedID: 0,

            //field data. Populated by setModalData with either blank data when adding a
            //new custom field or full data about an existing custom field when viewing/editing.
            fieldData: {} as customFieldDefined,

            //types of fields to choose from
            customFieldTypes: customFieldTypes,

            //need these for v-if in html
            customFieldTypeInteger: customFieldTypeInteger,
            customFieldTypeDecimal: customFieldTypeDecimal,
            customFieldTypeText: customFieldTypeText,
            customFieldTypeBoolean: customFieldTypeBoolean,
            customFieldTypeMultiChoice: customFieldTypeMultiChoice,
            customFieldTypeDate: customFieldTypeDate,

            separator: ";", //separator for multichoice options

            //errors
            submitting: false,
            msgSave: '',
            msgSaveType: '',

            //endpoints
            urls: {
                add: "/api/custom-fields/defined/add/",
                update: "/api/custom-fields/defined/update/",
                delete: "/api/custom-fields/defined/delete/",
            }
        },
        computed: {
            //adding is set to true when the user is adding a custom field. This is set
            //using the field's ID which is only set when looking up an existing field. This
            //is used to modify what the GUI displays (modal title and body text) to show 
            //correct helpful information based on adding or editing a field.
            adding: function () {
                if (this.fieldData.ID === undefined || this.fieldData.ID < 1) {
                    return true;
                }

                return false;
            },

            //multiChoiceOptions returns the user-provided semicolon separated options as an array
            //for building the select menu where a user can choose the default option.
            multiChoiceOptions: function () {
                //page is just loading, mounted hasn't been called yet to set default empty state
                if (this.fieldData.MultiChoiceOptions === undefined) {
                    return [];
                }

                //ignore when field isn't a multi choice
                if (this.fieldData.Type !== this.customFieldTypeMultiChoice) {
                    return [];
                }

                //no options provided by user yet
                if (this.fieldData.MultiChoiceOptions.trim().length === 0) {
                    return [];
                }

                //options provided, split string into array for use with v-for in HTML.
                let opsArray: string[] = this.fieldData.MultiChoiceOptions.split(this.separator);
                let validatedArray: string[] = [];
                for (let s of opsArray) {
                    s = s.trim();

                    //ignore blanks
                    if (s === "") {
                        continue;
                    }

                    //ignore duplicates
                    if (validatedArray.includes(s)) {
                        continue;
                    }

                    validatedArray.push(s);
                }

                return validatedArray;
            }
        },
        methods: {
            //setAppID sets the appSelectedID value in this vue object. This is called from
            //manageApps.setAppInOtherVueObjects() when an app is chosen from the list of
            //defined apps.
            setAppID: function (appID: number) {
                this.appSelectedID = appID;
                return;
            },

            //setModalData is used to populate the modal with data from the clicked field
            //in the list of custom fields. This is also used to reset the modal to a clean 
            //state when adding a new custom field.
            setModalData: function (item: customFieldDefined | undefined) {
                this.resetModal();

                //user wants to add a new custom field.
                if (item === undefined) {
                    return;
                }

                //user is viewing details of a custom field.
                this.fieldData = item

                //have to set stuff based on field type
                switch (item.Type) {
                    case this.customFieldTypeBoolean:
                        //@ts-ignore cannot find Vue
                        Vue.nextTick(function () {
                            setToggle("BoolDefaultValue", item.BoolDefaultValue);
                        });

                        break;
                }

                return;
            },

            //resetModal sets the modal back to a clean state for adding a new custom field.
            resetModal: function () {
                this.fieldData = {
                    ID: undefined,
                    DatetimeCreated: "",
                    DatetimeModified: "",
                    CreatedByUserID: 0,
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

            //setField saves a chosen radio toggle's value to the Vue object
            setField: function (fieldName: string, value: boolean) {
                this.fieldData[fieldName] = value;
                return;
            },

            //setToggleDefaultIfBool sets the toggle to the default false state when
            //the bool field type is chosen from the select menu. If we don't do this,
            //the toggle is unselected and even though app will default to false when 
            //saving we want to show the default.
            setToggleDefaultIfBool: function () {
                //@ts-ignore Vue not found
                Vue.nextTick(function () {
                    setToggle("BoolDefaultValue", modalCustomFieldDefined.fieldData.BoolDefaultValue);
                });

                return;
            },

            //addOrUpdate handles common validation before calling the correct function to
            //complete the api call.
            addOrUpdate: function () {
                //validation
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

                        //When validating multi choice options, we handle the provided options
                        //as an array. This allows for easier sanitizing and validating since we
                        //are going to display the options in a select from an array anyway.
                        let opsArray: string[] = this.fieldData.MultiChoiceOptions.split(this.separator);

                        //remove starting/ending whitespace from each option
                        //only put non-blank results into output array
                        //make sure element doesn't already exist in output array, warning if duplicate is found.
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

                        //make sure something exists in output array. this catch is user didn't
                        //provide any values or a blank value only.
                        if (validatedArray.length === 0) {
                            this.msgSave = "You must provide as least one option for this field.";
                            return;
                        }

                        //make sure default value is in list of options.
                        if (this.fieldData.MultiChoiceDefaultValue === undefined || this.fieldData.MultiChoiceDefaultValue.trim() === "") {
                            this.msgSave = "You must choose a default value for this field.";
                            return;
                        }
                        if (!validatedArray.includes(this.fieldData.MultiChoiceDefaultValue)) {
                            this.msgSave = "Please choose a default value from the list of options your provided.";
                            return;
                        }

                        //set validated array back
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

                //call correct function
                if (this.fieldData.ID !== undefined) {
                    this.update();
                }
                else {
                    this.add();
                }

                return;
            },

            //add saves a new field. The details in the form are submitting and the server
            //will create the new field and save it to this app's database. The modal will
            //then be reset to allow adding another field. The list of fields will also be 
            //updated (in parent card). This is called from addOrUpdate();
            add: function () {
                //make sure data isn't already being submitted
                if (this.submitting) {
                    console.log("already submitting...");
                    return;
                }

                //validation ok
                this.msgSave = "Adding...";
                this.msgSaveType = msgTypes.primary;
                this.submitting = true;

                //perform api call
                let data: Object = {
                    data: JSON.stringify(this.fieldData),
                };
                fetch(post(this.urls.add, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then(function (j) {
                        //check if response is an error from the server
                        let err: string = handleAPIErrors(j);
                        if (err !== '') {
                            modalCustomFieldDefined.msgSave = err;
                            modalCustomFieldDefined.msgSaveType = msgTypes.danger;
                            modalCustomFieldDefined.submitting = false;
                            return;
                        }

                        //Refresh the list of fields so that this new field is shown.
                        listCustomFieldsDefined.getCustomFields();

                        //Show success message briefly and reset the modal to an empty
                        //state so user can add another field.
                        modalCustomFieldDefined.msgSave = "Added!";
                        modalCustomFieldDefined.msgSaveType = msgTypes.success;
                        setTimeout(function () {
                            modalCustomFieldDefined.resetModal();
                            modalCustomFieldDefined.msgSave = '';
                            modalCustomFieldDefined.msgSaveType = '';
                            modalCustomFieldDefined.submitting = false;
                        }, defaultTimeout);

                        return;
                    })
                    .catch(function (err) {
                        console.log("fetch() error: >>", err, "<<");
                        modalCustomFieldDefined.msgSave = 'An unknown error occured. Please try again.';
                        modalCustomFieldDefined.msgSaveType = msgTypes.danger;
                        modalCustomFieldDefined.submitting = false;
                        return;
                    });

                return;
            },

            //update saves changes to an existing field. This is called from addOrUpdate().
            update: function () {
                //make sure data isn't already being submitted
                if (this.submitting) {
                    console.log("already submitting...");
                    return;
                }

                //Make sure we know what field we are updating.
                if (isNaN(this.fieldData.ID) || this.fieldData.ID === '' || this.fieldData.ID < 1) {
                    this.msgSave = "Could not determine which field you are trying to update. Please refresh the page and try again.";
                    this.msgSaveType = msgTypes.danger;
                    return;
                }

                //validation ok
                this.msgSave = "Saving...";
                this.msgSaveType = msgTypes.primary;
                this.submitting = true;

                //perform api call
                let data: Object = {
                    data: JSON.stringify(this.fieldData),
                };
                fetch(post(this.urls.update, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then(function (j) {
                        //check if response is an error from the server
                        let err: string = handleAPIErrors(j);
                        if (err !== '') {
                            modalCustomFieldDefined.msgSave = err;
                            modalCustomFieldDefined.msgSaveType = msgTypes.danger;
                            modalCustomFieldDefined.submitting = false;
                            return;
                        }

                        modalCustomFieldDefined.msgSave = "Changes saved!";
                        modalCustomFieldDefined.msgSaveType = msgTypes.success;
                        setTimeout(function () {
                            modalCustomFieldDefined.msgSave = '';
                            modalCustomFieldDefined.msgSaveType = '';
                            modalCustomFieldDefined.submitting = false;
                        }, defaultTimeout);

                        return;
                    })
                    .catch(function (err) {
                        console.log("fetch() error: >>", err, "<<");
                        modalCustomFieldDefined.msgSave = 'An unknown error occured. Please try again.';
                        modalCustomFieldDefined.msgSaveType = msgTypes.danger;
                        modalCustomFieldDefined.submitting = false;
                        return;
                    });

                return;
            },

            //remove marks a field as inactive. Inactive field will no long show up in
            //the list of field or be required when creating a new license.
            remove: function () {
                //make sure data isn't already being submitted
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

                //validation ok
                this.msgSave = "Deleting...";
                this.msgSaveType = msgTypes.primary;
                this.submitting = true;

                //perform api call
                let data: Object = {
                    id: this.fieldData.ID,
                };
                fetch(post(this.urls.delete, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then(function (j) {
                        //check if response is an error from the server
                        let err: string = handleAPIErrors(j);
                        if (err !== '') {
                            modalCustomFieldDefined.msgSave = err;
                            modalCustomFieldDefined.msgSaveType = msgTypes.danger;
                            return;
                        }

                        //refresh the list of custom fields in table. The modal will
                        //be closed by the data-dismiss on the button clicked.
                        listCustomFieldsDefined.getCustomFields();
                        modalCustomFieldDefined.submitting = false;

                        return;
                    })
                    .catch(function (err) {
                        console.log("fetch() error: >>", err, "<<");
                        modalCustomFieldDefined.msgSave = 'An unknown error occured. Please try again.';
                        modalCustomFieldDefined.msgSaveType = msgTypes.danger;
                        modalCustomFieldDefined.submitting = false;
                        return;
                    });
                return;
            },
        },
        mounted() {
            //this is used to set the object storing field data to a default state
            this.resetModal();

            return;
        }
    })
}