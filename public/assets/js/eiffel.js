document.addEventListener('DOMContentLoaded', initEiffel);
document.body.addEventListener('htmx:afterSettle', initEiffel);

let init = false;

async function initEiffel() {
    if (init) return;

    const elicitationForm = document.querySelector('.eiffel-elicitation')
    if (!elicitationForm) return;

    registerShortcuts();
    registerFocuses();
    registerCopyToClipboard();

    init = true;
}

function registerShortcuts() {
    // alt + f to open search modal - strg + f would collide with the browser search, people tend to hate when you override that :)
    document.addEventListener('keydown', function (event) {
        if (event.key === 'f' && event.altKey) {
            event.preventDefault();

            const searchBtn = document.querySelector('.eiffel-elicitation-template-search button');
            if (!searchBtn) return;

            searchBtn.click(); // showing the search modal directly will not trigger the hx-get on the button
        }
    });

    // alt + enter to submit elicitation form - we might as well keep the alt when we already use it everywhere else
    document.addEventListener('keydown', function (event) {
        if (event.key === 'Enter' && event.altKey) {
            event.preventDefault();

            const elicitationForm = document.getElementById('eiffelElicitationForm');
            if (!elicitationForm) return;

            elicitationForm.querySelector('button[type="submit"]').click(); // to trigger the hx-post on the button
        }
    });

    // alt + -> next variant - strg + -> would collide with selecting text in the inputs
    document.addEventListener('keydown', function (event) {
        if (event.key === 'ArrowRight' && event.altKey) {
            event.preventDefault();

            const current = document.getElementsByClassName('eiffel-template-variant-current')[0];
            if (!current) return;

            const next = document.getElementsByClassName('eiffel-template-variant-next')[0];
            if (!next) return;

            next.click();
        }
    });

    // alt + <- previous variant - strg + <- would collide with selecting text in the inputs
    document.addEventListener('keydown', function (event) {
        if (event.key === 'ArrowLeft' && event.altKey) {
            event.preventDefault();

            const current = document.getElementsByClassName('eiffel-template-variant-current')[0];
            if (!current) return;

            const previousItems = document.getElementsByClassName('eiffel-template-variant-prev');
            const previous = previousItems[previousItems.length - 1];
            if (!previous) return;

            previous.click();
        }
    });
}

function registerFocuses() {
    const searchModal = document.getElementById('eiffelTemplateSearch');
    if (!searchModal) console.error('could not find template search modal for elicitation');

    // focus search input when modal is shown
    searchModal.addEventListener('shown.bs.modal', function () {
        focusSearchInput();
    });

    // focus search input when search form is loaded (in the case bootstrap finishes showing the modal before the form is loaded)
    document.addEventListener('htmx:afterSettle', function(event) {
        if (event.detail.elt.id !== 'eiffelTemplateSearch') return;
        focusSearchInput();
    })

    // focus first input of elicitation form when modal is closed
    searchModal.addEventListener('hidden.bs.modal', function () {
        focusElicitationInput();
    });

    // focus first input of elicitation form when search form is loaded (in the case bootstrap finishes hiding the modal before the form is loaded)
    document.addEventListener('htmx:afterSettle', function(event) {
        if (event.detail.elt.id !== 'eiffelElicitationForm') return;
        focusElicitationInput();
    })
}

// focus search input inside search modal
function focusSearchInput() {
    const searchInput = document.getElementById('eiffelTemplateSearchInput');
    if (!searchInput) return;

    searchInput.focus();
}

// focus first input of elicitation form
function focusElicitationInput() {
    const firstInput = document.querySelector('#eiffelElicitationForm input:not([type="hidden"]):not([disabled]), #eiffelElicitationForm textarea:not([disabled])')
    if (!firstInput) return;

    // wait for the modal to be hidden/the form to be loaded - unfortunately, for firefox this is the only way it works
    // the 10ms timeout should be sufficient and is not noticeable
    setTimeout(function() {
        firstInput.focus();
    }, 10);
}

function registerCopyToClipboard() {
    document.addEventListener('htmx:afterSettle', function(event) {
        if (!event.detail.elt.className.includes('eiffel-elicitation-template-variant-form')) return;
        registerCopyBtn();

        // copy automatically if checkbox is checked
        const autoCopyCheckbox = document.getElementById('copyAfterParse');
        if (!autoCopyCheckbox || !autoCopyCheckbox.checked) return;

        const copyBtn = document.getElementsByClassName('eiffel-elicitation-form-copy-and-clear')[0];
        if (!copyBtn) return;

        copyBtn.click();
    });

    // alt + k to copy requirement to clipboard - this is not at random, but because alt + k is not used by any other shortcut
    // We don't want to override existing, important shortcuts. Also, alt + k is easy to reach and firefox will block copying
    // to the clipboard if it is not triggered by a user event (and it seems it doesn't like preventDefault in this context)
    document.addEventListener('keydown', function (event) {
        if (event.key === 'k' && event.altKey) {
            const copyBtn = document.getElementsByClassName('eiffel-elicitation-form-copy-and-clear')[0];
            if (!copyBtn) return;

            copyRequirementToClipboard().then(() => {
                clearElicitationForm();
                focusElicitationInput();
            });
        }
    });
}

function registerCopyBtn() {
    const copyBtn = document.getElementsByClassName('eiffel-elicitation-form-copy-and-clear')[0];
    if (!copyBtn) return;

    copyBtn.addEventListener('click', function () {
        copyRequirementToClipboard().then(() => {
            clearElicitationForm();
            focusElicitationInput();
        });
    });
}

function copyRequirementToClipboard() {
    const requirement = document.getElementsByClassName('eiffel-elicitation-form-requirement')[0];
    if (!requirement) return;

    return navigator.clipboard.writeText(requirement.value).catch(() => {
        alert('Unfortunately, your browser blocked copying the requirement to the clipboard. Please click somewhere on the page and try again then or try to copy manually. Using automatic copy + clear should work fine. Sorry for the inconvenience!')
    })
}

function clearElicitationForm() {
    const elicitationForm = document.getElementById('eiffelElicitationForm');
    if (!elicitationForm) return;

    const inputs = elicitationForm.querySelectorAll('input:not([type="hidden"]):not([disabled]), textarea:not([disabled])');
    inputs.forEach(input => {
        input.value = '';
    });
}