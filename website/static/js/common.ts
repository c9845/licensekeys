/**
 * common.ts
 * This file holds boilerplate code used elsewhere in the ts files.
 */

//version is the version number of the ts/js code in this app
//this isn't update very frequently and is really only used for debugging
const version: string = "3.0.1";

//defaultTimeout is the time used in setTimeout functions.  This is defined as a const
//so we always use the same time throughout the app
const defaultTimeout: number = 1500;

//msgTypes are the types of alerts from bootstrap that we use in the gui
const msgTypes: { [key: string]: string } = {
    danger: 'alert-danger',
    info: 'alert-info',
    success: 'alert-success',
    warning: 'alert-warning',
    primary: 'alert-primary',
};

//isEmail checks if a string could be an email address. This is just a very basic
//regex check.
/**
 * @param {string} text - The string to check for being an email
 * @returns {bool} - True if text is an email, false if not.
 */
function isEmail(text: string): boolean {
    let re: RegExp = /[a-z0-9!#$%&'*+/=?^_`{|}~-]+(?:\.[a-z0-9!#$%&'*+/=?^_`{|}~-]+)*@(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.)+[a-z0-9](?:[a-z0-9-]*[a-z0-9])?/;
    return re.test(text);
}

//enableSetToggleDebug is used to enable logging used in setToggle at runtime.
//Default value is false.
var enableSetToggleDebug = false;

//setToggle sets the correct button to "on" in a bootstrap btn-group where each button is a radio 
//This function is used because setting btn-group radio cannot be done easily through Vue.  Using this function is cleaner.
//Setting toggleToSetOn to undefined results in all toggle options in the btn-group being turned off.
//The "toggleToSetOn" should bet set as a data value like <input type="radio> data="true"> where true is the "toggleToSetOn".
//This function is used SET the state in the gui.
//To GET data from a btn-group, use setField() which should be defined on each Vue instance.
/**
 * @param {string} btnGroupElem - The ID of the element with the .btn-group class, the parent to input type=radio elements.
 * @param {string | boolean | undefined} toggleToSetIDOn - The option to set on.  This should be set in data- attribute.
 * @param {boolean} forClass - Optional field that allows setting toggles based on class, not an ID.
 */
function setToggle(querySelector: string, toggleToSetOn: string | boolean | undefined, forClass?: boolean) {
    //get button group element(s)
    //This is the parent div that encases multiple buttons and has the
    //.btn-group and .btn-group-toggle classes. Note that always handle
    //this as a NodeList just to make this function cleaner.
    let selector: string = "#" + querySelector;
    if (forClass) {
        selector = "." + querySelector
    }

    let btnGroupElems: NodeList = document.querySelectorAll(selector);

    if (enableSetToggleDebug) {
        console.log("**setToggle debugging**")
        console.log("  querySelector (given):     ", querySelector);
        console.log("  querySelector (understood):", selector);
        console.log("  num elements selected:     ", btnGroupElems.length);
        console.log("  toggle to set on:          ", toggleToSetOn);
    }

    //for each btn group element...set the btn groups to
    //an "off" state where each .btn and input is not "active"
    //or "checked". This resets the btn groups to a "clean" state
    //where we know everything is off.
    //@ts-ignore forEach does not exist on NodeList
    btnGroupElems.forEach(function (elem: HTMLElement) {
        //get the label.btn elements within the .btn-group
        //we have to remove the "active" class from these elements
        //label.btn elements are parents to input type=radio elements
        //that we need to remove the "checked" attribute from.
        let btns: NodeList = elem.querySelectorAll(".btn");
        //@ts-ignore forEach does not exist on NodeList
        btns.forEach(function (btn: HTMLElement) {
            //remove the "active" class
            btn.classList.remove("active");

            //choose the child input type=radio
            let radio: HTMLInputElement = btn.querySelector("input");

            //set the radio to "off"
            radio.checked = false;

            //if user just wants all toggles turned off, we are done with
            //this set of btn.
            if (toggleToSetOn === undefined) {
                return;
            }

            //turn on the specific toggle btn
            //Do this by checking if btn data- attribute matches the
            //user provided value.
            if (btn.dataset.switch === toggleToSetOn.toString()) {
                btn.classList.add("active");
                radio.checked = true;
            } else if (enableSetToggleDebug) {
                console.log("  mismatched data-switch:    ", btn.dataset.switch, toggleToSetOn);
            }

        }); //end forEach: loop through each btn element in a btn-group.

    }); //end forEach: loop through each btn-group element.

    return;
}

//todayPlus returns today's date plus a number of days in yyyy-mm-dd format. This is
//typically used for input values or min/max attributes.
/**
 * @param {number} n - The number of days to add to today. 
 * @returns - A date in yyyy-mm-dd format.
 */
function todayPlus(n: number) {
    let now: Date = new Date();
    let plus1: Date = new Date(now.setDate(now.getDate() + n));
    let y: number = plus1.getFullYear();
    let m: number = plus1.getMonth() + 1;
    let d: number = plus1.getDate();
    let yy: string = y.toString();
    let mm: string = (m < 10) ? "0" + m.toString() : m.toString();
    let dd: string = (d < 10) ? "0" + d.toString() : d.toString();

    return yy + "-" + mm + "-" + dd;
}

//dateAdd takes a yyyy-mm-dd formatted date, adds a number of days, and returns a
//date in yyyy-mm-dd format.
/**
 * @param {string} ymd - The date you want to add days to.
 * @param {number} days - The number of days to add to d.
 * @returns - d plus n days in yyyy-mm-dd format.
 */
function dateAdd(ymd: string, days: number): string {
    let split: string[] = ymd.split("-");
    let yIn: number = parseInt(split[0]);
    let mIn: number = parseInt(split[1]) - 1; //adjust month to zero based.
    let dIn: number = parseInt(split[2]);

    let input: Date = new Date(yIn, mIn, dIn);
    let added: Date = new Date(input.setDate(input.getDate() + days));

    let y: number = added.getFullYear();
    let m: number = added.getMonth() + 1;
    let d: number = added.getDate();
    let yy: string = y.toString();
    let mm: string = (m < 10) ? "0" + m.toString() : m.toString();
    let dd: string = (d < 10) ? "0" + d.toString() : d.toString();

    return yy + "-" + mm + "-" + dd;
}

//isValidDate checks if a provided input is a valid YYYY-MM-DD formatted date. This 
//is used to verify that user inputs to a <input type="date"> are full dates and that 
//a full four-digit year was provided.
//
//This should be used any time user is providing a date input.
/**
 * @param {string} dateString - An date string in YYYY-MM-DD format as read from an input's value.
 * @returns {boolean} - true if d is a yyyy-mm-dd data string, false if not.
 */
function isValidDate(dateString: string): boolean {
    //Split year, month, and day so we can check if each component of date has correct 
    //number of characters. This is done to catch instances where "21" is provided 
    //instead of "2021".
    let split: string[] = dateString.split("-");
    if (split.length !== 3) {
        return false;
    }

    let y: string = split[0];
    if (y.length !== 4) {
        return false;
    }
    if (parseInt(y) < 2000) {
        return false;
    }

    let m: string = split[1];
    if (m.length !== 2) {
        return false;
    }
    if (parseInt(m) < 1 || parseInt(m) > 12) {
        return false;
    }

    let d: string = split[2];
    if (d.length !== 2) {
        return false;
    }
    if (parseInt(d) < 1 || parseInt(d) > 31) {
        return false;
    }

    //Pass date string through regexp to catch any non-numeric values, formatting 
    //issues, or other errors that would make the date invalid.
    var regEx: RegExp = /^\d{4}-\d{2}-\d{2}$/;
    return dateString.match(regEx) != null;
}