const converter = new showdown.Converter();
let promptToRetry = null;
let uniqueIdToRetry = null;

const submitButton = document.getElementById('submit-button');
const regenerateResponseButton = document.getElementById('regenerate-response-button');
const promptInput = document.getElementById('prompt-input');
const attachmentInput = document.getElementById('attachment-input');
const attachmentList = document.getElementById('attachment-list');

const responseList = document.getElementById('response-list');
const fileInput = document.getElementById("whisper-file");

let isGeneratingResponse = false;
let loadInterval = null;
let selectedAttachments = [];
let attachmentsToRetry = [];
const maxAttachmentSize = 5 * 1024 * 1024;
const maxAttachmentCount = 8;

// Retrieve threadId from localStorage or initialize as null
let threadId = localStorage.getItem('threadId') || null;

promptInput.addEventListener('keydown', function (event) {
    if (event.key === 'Enter') {
        event.preventDefault();
        if (event.ctrlKey || event.shiftKey) {
            document.execCommand('insertHTML', false, '<br/><br/>');
        } else {
            getGPTResult();
        }
    }
});

function generateUniqueId() {
    const timestamp = Date.now();
    const randomNumber = Math.random();
    const hexadecimalString = randomNumber.toString(16);

    return `id-${timestamp}-${hexadecimalString}`;
}

function addResponse(selfFlag, prompt) {
    const uniqueId = generateUniqueId();
    const html = `
            <div class="response-container ${selfFlag ? 'my-question' : 'chatgpt-response'}">
                <img class="avatar-image" src="assets/img/${selfFlag ? 'me' : 'chatgpt'}.png" alt="avatar"/>
                <div class="prompt-content" id="${uniqueId}">${prompt}</div>
            </div>
        `;
    responseList.insertAdjacentHTML('beforeend', html);
    responseList.scrollTop = responseList.scrollHeight;
    return uniqueId;
}

function escapeHTML(value) {
    return String(value || '').replace(/[&<>"']/g, (char) => ({
        '&': '&amp;',
        '<': '&lt;',
        '>': '&gt;',
        '"': '&quot;',
        "'": '&#39;'
    }[char]));
}

function formatBytes(bytes) {
    if (!bytes) return '0 B';
    const units = ['B', 'KB', 'MB'];
    let size = bytes;
    let unit = 0;
    while (size >= 1024 && unit < units.length - 1) {
        size /= 1024;
        unit += 1;
    }
    return `${size.toFixed(size >= 10 || unit === 0 ? 0 : 1)} ${units[unit]}`;
}

function attachmentSummaryHTML(attachments) {
    if (!attachments?.length) return '';
    const items = attachments.map((attachment) => {
        const label = `${attachment.name} (${attachment.type || 'file'}, ${formatBytes(attachment.size)})`;
        if (attachment.previewUrl) {
            return `<li><img src="${attachment.previewUrl}" alt="">${escapeHTML(label)}</li>`;
        }
        return `<li><span class="attachment-file-badge">PDF</span>${escapeHTML(label)}</li>`;
    }).join('');
    return `<div class="sent-attachments"><strong>Attachments</strong><ul>${items}</ul></div>`;
}

function renderAttachmentList() {
    if (!attachmentList) return;
    attachmentList.replaceChildren();
    selectedAttachments.forEach((attachment) => {
        const chip = document.createElement('div');
        chip.className = 'attachment-chip';
        if (attachment.previewUrl) {
            const image = document.createElement('img');
            image.src = attachment.previewUrl;
            image.alt = '';
            chip.append(image);
        }
        const label = document.createElement('span');
        label.textContent = `${attachment.name} (${formatBytes(attachment.size)})`;
        chip.append(label);
        const remove = document.createElement('button');
        remove.type = 'button';
        remove.setAttribute('aria-label', `Remove ${attachment.name}`);
        remove.textContent = 'x';
        remove.addEventListener('click', () => {
            selectedAttachments = selectedAttachments.filter((item) => item.id !== attachment.id);
            renderAttachmentList();
        });
        chip.append(remove);
        attachmentList.append(chip);
    });
}

function readAttachment(file) {
    return new Promise((resolve, reject) => {
        const isImage = file.type.startsWith('image/');
        const isPDF = file.type === 'application/pdf';
        if (!isImage && !isPDF) {
            reject(new Error(`${file.name} is not a supported PDF or image file.`));
            return;
        }
        if (file.size > maxAttachmentSize) {
            reject(new Error(`${file.name} is larger than 5 MB.`));
            return;
        }
        const attachment = {
            id: generateUniqueId(),
            name: file.name,
            type: file.type || 'application/octet-stream',
            size: file.size
        };
        if (!isImage) {
            resolve(attachment);
            return;
        }
        const reader = new FileReader();
        reader.addEventListener('load', () => {
            attachment.previewUrl = reader.result;
            resolve(attachment);
        });
        reader.addEventListener('error', () => reject(new Error(`Could not read ${file.name}.`)));
        reader.readAsDataURL(file);
    });
}

function loader(element) {
    element.textContent = '';

    loadInterval = setInterval(() => {
        // Update the text content of the loading indicator
        element.textContent += '.';

        // If the loading indicator has reached three dots, reset it
        if (element.textContent === '....') {
            element.textContent = '';
        }
    }, 300);
}

function setErrorForResponse(element, message) {
    element.innerHTML = message;
    element.style.color = 'rgb(200, 0, 0)';
}

function setRetryResponse(prompt, uniqueId, attachments = []) {
    promptToRetry = prompt;
    uniqueIdToRetry = uniqueId;
    attachmentsToRetry = attachments;
    regenerateResponseButton.style.display = 'flex';
}

async function regenerateGPTResult() {
    try {
        await getGPTResult(promptToRetry, uniqueIdToRetry);
        regenerateResponseButton.classList.add("loading");
    } finally {
        regenerateResponseButton.classList.remove("loading");
    }
}

async function getWhisperResult() {
    if (!fileInput?.files?.length) {
        return;
    }
    const formData = new FormData();
    formData.append("audio", fileInput.files[0]);
    const uniqueId = addResponse(false);
    const responseElement = document.getElementById(uniqueId);
    isGeneratingResponse = true;
    loader(responseElement);

    try {
        submitButton.classList.add("loading");
        const response = await fetch("/transcribe", {
            method: "POST",
            body: formData
        });
        if (!response.ok) {
            setErrorForResponse(responseElement, `HTTP Error: ${await response.text()}`);
            return;
        }
        const responseText = await response.text();
        responseElement.innerHTML = `<div>${responseText}</div>`;
    } catch (e) {
        console.log(e);
        setErrorForResponse(responseElement, `Error: ${e.message}`);
    } finally {
        isGeneratingResponse = false;
        submitButton.classList.remove("loading");
        clearInterval(loadInterval);
    }
}

// Function to get GPT result with streaming updates
async function getGPTResult(_promptToRetry, _uniqueIdToRetry) {
    const prompt = (_promptToRetry ?? promptInput.textContent).trim();
    const attachments = _uniqueIdToRetry ? attachmentsToRetry : selectedAttachments;

    if (isGeneratingResponse || (!prompt && !attachments.length)) {
        return;
    }

    submitButton.classList.add("loading");
    promptInput.textContent = '';
    selectedAttachments = [];
    renderAttachmentList();

    if (!_uniqueIdToRetry) {
        addResponse(true, `<div>${escapeHTML(prompt)}</div>${attachmentSummaryHTML(attachments)}`);
    }

    const uniqueId = _uniqueIdToRetry ?? addResponse(false);
    const responseElement = document.getElementById(uniqueId);
    loader(responseElement);
    isGeneratingResponse = true;

    try {
        const response = await fetch("/api/chatgpt-client/get-prompt-result", {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ prompt, threadId, attachments }), // Send threadId and attachments with the prompt
        });

        if (!response.ok) {
            throw new Error(`HTTP Error: ${await response.text()}`);
        }

        // Save the threadId from the response headers
        const responseThreadId = response.headers.get('Thread-ID');
        if (responseThreadId && responseThreadId !== threadId) {
            threadId = responseThreadId;
            localStorage.setItem('threadId', threadId); // Save threadId to localStorage
        }

        // Stream response and update UI incrementally
        const reader = response.body.getReader();
        const decoder = new TextDecoder();
        responseElement.innerHTML = '';
        let readDone, chunk;
        let responseText = '';

        while (!readDone) {
            ({ done: readDone, value: chunk } = await reader.read());
            if (chunk) {
                responseText += decoder.decode(chunk);
                responseElement.innerHTML = converter.makeHtml(formatMessage(responseText));
                responseList.scrollTop = responseList.scrollHeight;
            }
        }

        promptToRetry = null;
        uniqueIdToRetry = null;
        attachmentsToRetry = [];
        regenerateResponseButton.style.display = 'none';
        setTimeout(() => {
            responseList.scrollTop = responseList.scrollHeight;
            hljs.highlightAll();
        }, 10);
    } catch (err) {
        setRetryResponse(prompt, uniqueId, attachments);
        setErrorForResponse(responseElement, `Error: ${err.message}`);
    } finally {
        isGeneratingResponse = false;
        submitButton.classList.remove("loading");
        clearInterval(loadInterval);
    }
}

attachmentInput?.addEventListener('change', async () => {
    const files = Array.from(attachmentInput.files || []);
    attachmentInput.value = '';
    for (const file of files) {
        if (selectedAttachments.length >= maxAttachmentCount) {
            window.alert(`You can attach up to ${maxAttachmentCount} files per message.`);
            break;
        }
        try {
            selectedAttachments.push(await readAttachment(file));
        } catch (error) {
            window.alert(error.message);
        }
    }
    renderAttachmentList();
});

submitButton.addEventListener("click", () => {
    getGPTResult();
});

regenerateResponseButton.addEventListener("click", () => {
    regenerateGPTResult();
});

document.addEventListener("DOMContentLoaded", function () {
    promptInput.focus();
});

function formatMessage(message) {
    // Convert phone numbers to clickable links
    const phoneRegex = /(\+\d{1,2}\s?)?(\(?\d{3}\)?[-.\s]?){2,3}\d{4}/g;
    message = message.replace(phoneRegex, (phone) => `<a href="tel:${phone}" style="color: #10A37F;">${phone}</a>`);

    // Convert URLs to clickable links
    const urlRegex = /(https?:\/\/[^\s]+)/g; // For absolute URLs
    message = message.replace(urlRegex, (url) => `<a href="${url}" target="_blank" style="color: #10A37F;">${url}</a>`);

    const markdownUrlRegex = /\[([^\]]+)\]\((https?:\/\/[^\s]+)\)/g; // For markdown-style links
    message = message.replace(markdownUrlRegex, (match, text, url) => `<a href="${url}" target="_blank" style="color: #10A37F;">${text}</a>`);

    return message;
}
