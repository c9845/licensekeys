/*
This file is used to automate minification of js and css files.
This ties into vscode task runner (tasks.json).
*/

//This creates the funcs (uglify, cssmin, watch).
module.exports = function(grunt) {
    grunt.initConfig({
        //minify js
        //This takes the script.js file created by the typescript compiler and creates
        //the minified version, script.min.js.  The script.js file is created by concatting
        //the *.ts files together (with some smarts behind it for ts references and ts checking).
        uglify: {
            app: {
                src:  './website/static/js/script.js',
                dest: './website/static/js/script.min.js',

            },
        },

        //minify css
        //This takes the styles.css file we modify and creates the minified version, styles.min.css.
        cssmin: {
            app: {
                src:  './website/static/css/styles.css',
                dest: './website/static/css/styles.min.css',
            },
        },

        //watch for changes on non-minified files
        //This watches the script.js and styles.css files for changes and when changes are noticed runs
        //the minifiers.  
        //This only runs the minified for the file type that has changed.  Aka, if js is changed, only run
        //the js minifier.  This makes the grunt/minification a bit faster since we aren't minifining files
        //that haven't changed. This also only runs the minifiers on the correct subdirectory (sales pages)
        //if needed to keep minifying fast(er).
        watch: {
            watchJSApp: {
                files: ['./website/static/js/script.js'],
                tasks: ['uglify'],
            },

            watchCSSApp: {
                files: ['./website/static/css/styles.css'],
                tasks: ['cssmin:app'],
            },
        },
    });

    //Load the task that were defined.
    grunt.loadNpmTasks('grunt-contrib-uglify');
    grunt.loadNpmTasks('grunt-contrib-cssmin');
    grunt.loadNpmTasks('grunt-contrib-watch');

    //Run this task when grunt starts.
    grunt.registerTask('default', ['watch']);
};