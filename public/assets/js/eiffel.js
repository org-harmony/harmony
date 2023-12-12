const EiffelMaxRequirementsInLocalStorage = 420;

document.addEventListener('DOMContentLoaded', registerDynamicFocuses);
document.addEventListener('htmx:afterSettle', registerDynamicFocuses);

document.addEventListener('DOMContentLoaded', initRequirementsList);
document.addEventListener('htmx:afterSettle', initRequirementsList);

document.addEventListener('DOMContentLoaded', autoResizeInput);
document.addEventListener('htmx:afterSettle', autoResizeInput);

document.addEventListener('DOMContentLoaded', registerOutputEmptyBtn);
document.addEventListener('htmx:afterSettle', registerOutputEmptyBtn);

document.addEventListener('htmx:afterRequest', requirementParsed);
document.addEventListener('newRequirementEvent', newRequirement);
document.addEventListener('emptyRequirementsEvent', emptyRequirements);

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

function registerOutputEmptyBtn() {
    const outputEmptyBtn = document.getElementById('eiffelRequirementsEmpty');
    if (!outputEmptyBtn || outputEmptyBtn.dataset.eiffelStatus === 'setup') return;

    outputEmptyBtn.addEventListener('click', function () {
        document.dispatchEvent(new CustomEvent('emptyRequirementsEvent'));
    });

    outputEmptyBtn.dataset.eiffelStatus = 'setup';
}

function copyOutputToClipboard(event) {
    const target = event.target;
    if (!target) return;

    const output = target.innerText;

    return navigator.clipboard.writeText(output)
        .catch(() => {
            alert('Sorry, your browser blocked copying the output to the clipboard. Try to copy manually.');
        });
}

function clearElicitationForm() {
    const elicitationForm = document.getElementById('eiffelElicitationForm');
    if (!elicitationForm) return;

    const inputs = elicitationForm.querySelectorAll('input:not([type="hidden"]):not([disabled]), textarea:not([disabled])');
    inputs.forEach(input => {
        input.value = '';
        input.dispatchEvent(new Event('change'));
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

        const setHeight = () => {
            input.style.height = 'auto';
            input.style.height = input.scrollHeight + 'px';
        }

        setHeight();
        input.addEventListener('input', setHeight);
        input.addEventListener('change', setHeight);

        input.dataset.eiffelStatus = 'setup';
    });
}

function requirementParsed(event) {
    const xhr = event.detail.xhr;
    if (!xhr) return;

    const responseHeaders = xhr.getResponseHeader('ParsingSuccessEvent');
    if (!responseHeaders) return;

    // base64 decode the response header and parse it as JSON
    event = JSON.parse(new TextDecoder().decode(base64ToBytes(responseHeaders)));
    if (!event) return;

    const parsingSuccessEvent = event.parsingSuccessEvent;
    if (!parsingSuccessEvent) return;

    const requirement = parsingSuccessEvent.requirement;
    if (!requirement) return;

    const timestamp = Date.now();
    let key = `eiffel-requirement-${timestamp}`;
    localStorage.setItem(key, requirement);

    document.dispatchEvent(new CustomEvent('newRequirementEvent', {
        detail: {
            requirement: requirement,
            key: key
        }
    }));
}

function newRequirement(event) {
    const requirement = event.detail.requirement;
    const key = event.detail.key;
    if (!requirement) return;

    const requirementList = document.querySelector('#eiffelRequirementsListWrapper ul');
    if (!requirementList) return;

    const firstListItem = requirementList.querySelector('ul > li.eiffel-requirements-list-item')
    if (!firstListItem) return;

    const newListItem = firstListItem.cloneNode(true);
    newListItem.innerText = requirement;
    newListItem.dataset.eiffelRequirementKey = key;
    newListItem.addEventListener('click', copyOutputToClipboard);
    requirementList.prepend(newListItem);

    if (!firstListItem.dataset.eiffelRequirementKey) {
        firstListItem.classList.add('d-none');
    }
}

function initRequirementsList() {
    const requirementListWrapper = document.getElementById('eiffelRequirementsListWrapper');
    if (!requirementListWrapper || requirementListWrapper.dataset.eiffelStatus === 'setup') return;

    let items = {};

    // get all items from local storage
    for (let i = 0; i < localStorage.length; i++) {
        const key = localStorage.key(i);
        if (!key.startsWith('eiffel-requirement-')) continue;

        const requirement = localStorage.getItem(key);
        if (!requirement) continue;

        items[key] = requirement;
    }

    // sort items ascending by timestamp (from key)
    const sortedKeys = Object.keys(items).sort((a, b) => {
        const aTimestamp = parseInt(a.replace('eiffel-requirement-', ''));
        const bTimestamp = parseInt(b.replace('eiffel-requirement-', ''));

        return aTimestamp - bTimestamp;
    });
    const sortedItems = {};
    sortedKeys.forEach(key => {
        sortedItems[key] = items[key];
    });
    items = sortedItems;

    // clean up old items
    items = cleanRequirementsList(items, EiffelMaxRequirementsInLocalStorage);

    // call newRequirement for each item
    Object.keys(items).forEach(key => {
        document.dispatchEvent(new CustomEvent('newRequirementEvent', {
            detail: {
                requirement: items[key],
                key: key
            }
        }));
    });

    requirementListWrapper.dataset.eiffelStatus = 'setup';
}

// cleanup oldest items if there are more than max items
// expects items to be an object with key => value pairs that is sorted ascending by timestamp (from key)
// returns the cleaned items object that was passed in
function cleanRequirementsList(items, max) {
    const keys = Object.keys(items);
    // return early if there are less than max items
    if (keys.length <= max) return items;

    // delete the oldest items
    const keysToDelete = keys.slice(0, keys.length - max);
    keysToDelete.forEach(key => {
        localStorage.removeItem(key);
    });
    console.info(`Removed ${keysToDelete.length} requirements from local storage.`);

    // remove the deleted items from the list
    keysToDelete.forEach(key => {
        delete items[key];
    });

    return items;
}

function emptyRequirements() {
    const requirementList = document.querySelector('#eiffelRequirementsListWrapper ul');
    if (!requirementList) return;

    const itemNodes = requirementList.querySelectorAll('li.eiffel-requirements-list-item');
    if (!itemNodes) return;

    const items = {};
    itemNodes.forEach(itemNode => {
        if (!itemNode.dataset.eiffelRequirementKey) return;
        items[itemNode.dataset.eiffelRequirementKey] = itemNode.innerText;
    });

    cleanRequirementsList(items, 0);

    itemNodes.forEach((itemNode) => {
        if (!itemNode.dataset.eiffelRequirementKey) {
            itemNode.classList.remove('d-none');
            return;
        }

        itemNode.remove();
    });
}

function base64ToBytes(base64) {
    const binString = atob(base64);
    return Uint8Array.from(binString, (m) => m.codePointAt(0));
}