/**
 * common.ts
 * 
 * This file holds boilerplate code used elsewhere in the ts files.
 */

//version is the version number of the ts/js code. This may not match the app's 
//version if the ts/js code has not changed.
export const version: string = "4.0.0";
console.log("JS Version:", version);

//Set JS version in GUI if element exists. AKA, on the diagnostics page. Setting the
//version is done this way, instead of in <script> tag like other apps, because this
//app used JS modules which means the version const isn't accessible in <script> tags.
if (document.getElementById("js-version")) {
    document.getElementById("js-version")!.innerHTML = version;
}

//apiBaseURL is the path API endpoints are based off of.
export const apiBaseURL: string = "/api/";


//defaultTimeout is the time used in setTimeout functions. This is defined as a const 
//so we always use the same time throughout the app
export const defaultTimeout: number = 1500;

//msgTypes are the types of alerts from bootstrap that we use in the GUI. These 
//values are used on "alert" divs.
export const msgTypes: { [key: string]: string } = {
    danger: 'alert-danger',
    info: 'alert-info',
    success: 'alert-success',
    warning: 'alert-warning',
    primary: 'alert-primary',
};

//isEmail checks if a string could be an email address. This is just a very basic 
//regex check. This is used for checking email addresses during login or when adding
//a user.
/**
 * @param {string} text - The string to check for being an email.
 * @returns {bool} - True if text is an email, false if not.
 */
export function isEmail(text: string): boolean {
    let re: RegExp = /[a-z0-9!#$%&'*+/=?^_`{|}~-]+(?:\.[a-z0-9!#$%&'*+/=?^_`{|}~-]+)*@(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.)+[a-z0-9](?:[a-z0-9-]*[a-z0-9])?/;
    return re.test(text);
}


//getToday returns today's date in yyyy-mm-dd format.
export function getToday() {
    let now: Date = new Date();
    let y: number = now.getFullYear();
    let m: number = now.getMonth() + 1;
    let d: number = now.getDate();
    let yy: string = y.toString();
    let mm: string = (m < 10) ? "0" + m.toString() : m.toString();
    let dd: string = (d < 10) ? "0" + d.toString() : d.toString();

    return yy + "-" + mm + "-" + dd;
}

//todayPlus returns today's date plus a number of days in yyyy-mm-dd format. This is 
//typically used for input values or min/max attributes.
/**
 * @param {number} n - The number of days to add to today. 
 * @returns - A date in yyyy-mm-dd format.
 */
export function todayPlus(n: number) {
    let now: Date = new Date();
    let plus: Date = new Date(new Date().setDate(now.getDate() + n));
    let y: number = plus.getFullYear();
    let m: number = plus.getMonth() + 1;
    let d: number = plus.getDate();
    let yy: string = y.toString();
    let mm: string = (m < 10) ? "0" + m.toString() : m.toString();
    let dd: string = (d < 10) ? "0" + d.toString() : d.toString();

    return yy + "-" + mm + "-" + dd;
}

//isValidDate checks if a provided input is a valid YYYY-MM-DD formatted date. This is
//used to verify that user inputs to a <input type="date"> are full dates and that a full
//four-digit year was provided.
//
//This should be used any time user is providing a date input.
/**
 * @param {string} dateString - An date string in YYYY-MM-DD format as read from an input's value.
 * @returns {boolean} - true if d is a yyyy-mm-dd data string, false if not.
 */
export function isValidDate(dateString: string): boolean {
    //Split year, month, and day so we can check if each component of date has correct number
    //of characters. This is done to catch instances where "21" is provided instead of "2021".
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

    //Pass date string through regexp to catch any non-numeric values, formatting issues, or
    //other errors that would make the date invalid.
    var regEx: RegExp = /^\d{4}-\d{2}-\d{2}$/;
    return dateString.match(regEx) != null;
}
