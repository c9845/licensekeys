/**
 * fetch.ts
 * This file holds functions to help deal with fetch() requests.
 */

//ResponseData is the format of the data that is sent back from the api
//data can hold anything but is usually an object or an array of object
interface ResponseData {
    OK:         boolean;
    Type:       string; //dataFound, insertOK, updateOK, etc.
    Datetime:   string; //YYYY-MM-DD HH:MM:SS.sssZ UTC
    
    //only one of these should be set
    //Data should only be set if OK is true.
    //ErrorData should only be set if OK is false.
    Data?:      any;
    ErrorData?: ErrorData;
}

//ErrorData is the format of the data that is detail of the error that occured in an api call
interface ErrorData {
    Error:      string; //simple error message more for diagnostics.
    Message:    string; //a human readable and displayable message.
}

//objectToString converts an object into a url encoded string for submitting in a form.
//This is done to place nicely with golang's r.FormValue() func to reading values and
//for parsing of JSON values. This is nicer then sending up a JSON encoded request body
//which would need to be parsed on the golang server and would require a specific struct
//for each endpoint depending on what data in the JSON payload was sent up.
//This func returns a string in the form of key=value&key2=value2,...
function objectToString(obj: Object): string {
    var params = Object.keys(obj).map(function(key) {
        return key + '=' + encodeURIComponent(obj[key]);
    }).join('&');

    return params;
}

//removeDoubleSlash removes a double slash in a url replacing it with a single
//slash. Double slashes are a typo mistake and should be fixed.
function removeDoubleSlash(url: string): string {
    //check if a double slash exists to show a warning
    if (url.includes("//")) {
        console.log("URL contains a double slash, this should be fixed.", url);
    }

    //fix double slash
    url = url.replace(/\/\//g, "/");
    return url;
}

//get builds a get request to use in fetch().
function get(url: string, formValues: Object): Request {
    //make sure url doesn't have double slashes by mistake
    url = removeDoubleSlash(url);

    //build the request
    let r: RequestInit = {
        method:         'GET',
        credentials:    'same-origin',
        headers:{
            "Content-type": "application/x-www-form-urlencoded; charset=UTF-8"
        },
    };

    //get the form values as a string for use in appending to url
    let params: string = objectToString(formValues);

    //build the url by adding the form values to the end of it.
    //need the ? separator between url and parameters.
    //Cannot use r.body like in a POST request.
    url = url + "?" + params;

    //return the object to use in fetch request
    return new Request(url, r);
}

//post build a post request to use in fetch()
function post(url: string, formValues: Object): Request {
    //make sure url doesn't have double slashes by mistake
    url = removeDoubleSlash(url);

    //build the request
    let r: RequestInit = {
        method:         'POST',
        credentials:    'same-origin',
        headers:{
            "Content-type": "application/x-www-form-urlencoded; charset=UTF-8"
        },
    };

    //get the form values as a string to use in the request body
    let params: string = objectToString(formValues);

    //add the form values to the request body
    //We don't append the form values to the url, like in a GET request.
    r.body = params;

    //return the object to use in fetch request
    return new Request(url, r);
}

//postFile build a post request to use in fetch() to upload a file
//don't need Content-Type header as it will be set automatically.
//dont need to handle building form value params as it will be handled automatically.
function postFile(url: string, fileData: FormData): Request {
    //make sure url doesn't have double slashes by mistake
    url = removeDoubleSlash(url);

    //build the request config
    let r: RequestInit = {
        method:         'POST',
        credentials:    'same-origin',
        body:           fileData
    };

    //return the object to use in fetch request
    return new Request(url, r);
}

//handleRequestErrors handles errors that occur when making the fetch() request
//this could occur when a server is down, network issues, etc.
//this does *not* handle errors returned from the server such as validation errors or other server issues
//our server should only return:
// - 200 when requests complete successfully
// - 500 when an error occured on the server (validation issue, db issue, etc.)
//Handle any other status codes (page not found, server unavailable, network issue, etc) via .catch().
function handleRequestErrors(response: Response): Response {
    let serverResponseCodes: number[] = [200, 500];
    if (serverResponseCodes.indexOf(response.status) == -1) {
        //console.log("fetch request error: bad status");
        //fetch().catch will handle this...
        //@ts-ignore promise only refers to type but is being used as a value
        return Promise.reject(new Error(response.statusText));
    }
    
    //everything ok, move to next promise
    //@ts-ignore promise only refers to type but is being used as a value
    return Promise.resolve(response)
}

//getJSON gets the response body for a fetch() request as json which is how our golang server
//send back all response data
function getJSON(response: Response): ResponseData {
    //@ts-ignore
    let j: ResponseData = (response.json() as unknown as ResponseData)
    return j
}

//handleAPIErrors checks if an api call resulted in an error and responds with
//an error message to show to the user
//if there isn't an error, this responds with a blank string
function handleAPIErrors(j: ResponseData): string {
    //check if an error occured and the error can be displayed in the gui
    if (j.OK === false) {
        return j.ErrorData.Message;
    }

    //no error
    //return nothing
    return '';
}