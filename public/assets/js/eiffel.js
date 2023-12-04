document.addEventListener('DOMContentLoaded', registerDynamicFocuses);
document.addEventListener('htmx:afterSettle', registerDynamicFocuses);

document.addEventListener('DOMContentLoaded', autoResizeInput);
document.addEventListener('htmx:afterSettle', autoResizeInput);

registerFocuses();

registerShortcuts();

registerCopyToClipboard();

function registerFocuses() {
    // focus search input when search form is loaded (in the case bootstrap finishes showing the modal before the form is loaded)
    document.addEventListener('htmx:afterSettle', function(event) {
        if (event.detail.elt.id !== 'eiffelTemplateSearch') return;
        focusSearchInput();
    })

    // focus first input of elicitation form when elicitation form is loaded (in case htmx finishes after bootstrap)
    document.addEventListener('htmx:afterSettle', function(event) {
        if (event.detail.elt.id !== 'eiffelElicitationTemplate') return;
        focusElicitationInput();
    })

    // focus first input of elicitation when output file form was loaded (in case htmx finishes after bootstrap)
    document.addEventListener('htmx:afterSettle', function(event) {
        if (!event.detail.elt.classList.contains('eiffel-requirements')) return;
        focusElicitationInput();
    })
}

function registerCopyToClipboard() {
    document.addEventListener('htmx:afterSettle', async function(event) {
        if (!event.detail.elt.className.includes('eiffel-elicitation-template-variant-form')) return;
        registerCopyBtn();

        // copy automatically if checkbox is checked
        const autoCopyCheckbox = document.getElementById('copyAfterParse');
        if (!autoCopyCheckbox || !autoCopyCheckbox.checked) return;

        await copyRequirementToClipboard();
    });
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

    // alt + o to focus output dir + file input
    document.addEventListener('keydown', function (event) {
        if (event.key === 'o' && event.altKey) {
            event.preventDefault();

            const outputDirInput = document.getElementById('eiffelOutputDir');
            if (!outputDirInput) return;

            outputDirInput.focus();
        }
    });

    // alt + p to focus requirement parsing form (first input)
    document.addEventListener('keydown', function (event) {
        if (event.key === 'p' && event.altKey) {
            event.preventDefault();
            focusElicitationInput();
        }
    });

    // alt + k to copy requirement to clipboard - this is not at random, but because alt + k is not used by any other shortcut
    // We don't want to override existing, important shortcuts. Also, alt + k is easy to reach and firefox will block copying
    // to the clipboard if it is not triggered by a user event (and it seems it doesn't like preventDefault in this context)
    document.addEventListener('keydown', async function (event) {
        if (event.key === 'k' && event.altKey) {
            const copyBtn = document.getElementsByClassName('eiffel-elicitation-form-copy-and-clear')[0];
            if (!copyBtn) return;

            await copyRequirementToClipboard();
        }
    });
}

function registerDynamicFocuses() {
    const templateSearch = document.getElementById('eiffelTemplateSearch');
    if (!templateSearch || templateSearch.dataset.eiffelStatus === 'setup') return;

    // focus search input inside search modal when modal is shown
    templateSearch.addEventListener('shown.bs.modal', function () {
        focusSearchInput();
    });

    // focus first input of elicitation form when modal is closed
    templateSearch.addEventListener('hidden.bs.modal', function () {
        focusElicitationInput();
    });

    templateSearch.dataset.eiffelStatus = 'setup';
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

    setTimeout(() => {
        firstInput.focus();
    }, 7); // wait for the modal to be fully closed - unfortunately, this is necessary
}

function registerCopyBtn() {
    const copyBtn = document.getElementsByClassName('eiffel-elicitation-form-copy-and-clear')[0];
    if (!copyBtn || copyBtn.dataset.eiffelStatus === 'setup') return;

    copyBtn.addEventListener('click', async function () {
        await copyRequirementToClipboard();
    });

    copyBtn.dataset.eiffelStatus = 'setup';
}

function copyRequirementToClipboard() {
    const requirement = document.getElementsByClassName('eiffel-elicitation-form-requirement')[0];
    if (!requirement) return;

    return navigator.clipboard.writeText(requirement.value)
        .catch(() => {
            alert('Unfortunately, your browser blocked copying the requirement to the clipboard. Please click somewhere on the page and try again then or try to copy manually. Sorry for the inconvenience!');
        })
        .then(() => {
            clearElicitationForm();
        })
        .finally(() => {
            focusElicitationInput();
        });
}

function clearElicitationForm() {
    const elicitationForm = document.getElementById('eiffelElicitationForm');
    if (!elicitationForm) return;

    const inputs = elicitationForm.querySelectorAll('input:not([type="hidden"]):not([disabled]), textarea:not([disabled])');
    inputs.forEach(input => {
        input.value = '';
    });
}

function autoResizeInput() {
    const inputs = document.querySelectorAll('[data-eiffel-auto-resize]');
    inputs.forEach(input => {
        if (input.dataset.eiffelStatus === 'setup') return;

        // border-box is important to include the padding and border in the height
        input.style.boxSizing = 'border-box';
        // disable resizing by the user
        input.style.resize = 'none';

        // init height for cases of autofill
        input.style.height = 'auto';
        input.style.height = input.scrollHeight + 'px';

        input.addEventListener('input', function () {
            input.style.height = 'auto';
            input.style.height = input.scrollHeight + 'px';
        });

        input.dataset.eiffelStatus = 'setup';
    });
}