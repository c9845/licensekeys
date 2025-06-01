/**
 * bs-tooltip.ts
 * 
 * This file holds the code to get Bootstrap tooltips to work with Vue populated data 
 * in the GUI. The standard bootstrap method of setting the title attribute and 
 * calling the function to register tooltips doesn't work since Vue loads the data 
 * after the GUI is built.
 */

const bsTooltip = (el: HTMLElement, binding: any) => {
    const t = []

    if (binding.modifiers.focus) t.push('focus')
    if (binding.modifiers.hover) t.push('hover')
    if (binding.modifiers.click) t.push('click')
    if (!t.length) t.push('hover')

    if (binding.value === undefined) {
        //console.log("bs-tooltip.ts - Value is undefined", el)
        return;
    }

    //This was added to support the data-html attribute as defined by
    //bootstrap. It allows for html tags in v-tooltip declarations.
    let useHTML = false;
    if (el.dataset.html) {
        useHTML = true;
    }

    //@ts-ignore tooltip doesn't exist
    $(el).tooltip({
        title: binding.value,

        //https://vuejs.org/guide/reusability/custom-directives.html#:~:text=arg%3A%20The%20argument%20passed%20to%20the%20directive%2C%20if%20any.%20For%20example%20in%20v%2Dmy%2Ddirective%3Afoo%2C%20the%20arg%20would%20be%20%22foo%22.
        placement: binding.arg || 'top',

        trigger: t.join(' '),
        html: useHTML,
    });
}

//@ts-ignore cannot find name vue
Vue.directive('tooltip', {
    bind: bsTooltip,
    update: bsTooltip,
    unbind(el: HTMLElement) {
        //@ts-ignore tooltip doesn't exist
        $(el).tooltip('dispose')
    }
});
