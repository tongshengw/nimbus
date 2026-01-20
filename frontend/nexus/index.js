let apiBase = "https://vmapi.tongshengwu.com"

document.addEventListener("DOMContentLoaded", async () => {
    await fetch(apiBase + "/check-status", {
        method: "GET",
    }).then(response => {
        if (!response.ok) {
            throw new Error("response err");
        }
        return response.json();
    }).then(data => {
        if (data['status'] == 'ok') {
            const button = document.querySelector('.get-machine-button');
            if (button) {
                button.disabled = false;
                button.textContent = 'Get Machine';
            }
        }
    });
});

async function getNewMachine() {
    const button = document.querySelector('.get-machine-button');
    if (button) {
        button.disabled = true;
        button.textContent = 'Loading...';
    }

    let newMachineResponse;
    await fetch(apiBase + "/new-machine", {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
    }).then(response => {
        if (!response.ok) {
            throw new Error("response err");
        }
        return response.json();
    }).then(data => {
        newMachineResponse = data;
    });

    console.log(newMachineResponse)

    const response = await fetch(apiBase + "/private/ssh-key", {
        method: "GET",
        headers: {
            "Authorization": newMachineResponse["token"]
        },
    });

    const blob = await response.blob();
    const blobUrl = URL.createObjectURL(blob);

    const a = document.createElement('a');
    a.href = blobUrl;
    a.download = newMachineResponse["machine_name"] + "_ssh_key"
    document.body.appendChild(a);
    a.style.display = 'none';
    a.click();
    a.remove();
    URL.revokeObjectURL(blobUrl);

    button.textContent = "Done!";

    const codeSnippetContainer = document.querySelector('.code-snippet-container');
    if (codeSnippetContainer) {
        codeSnippetContainer.style.display = 'flex';
    }
    const codeSnippetText = document.getElementById('id-code-snippet-text');
    if (codeSnippetText) {
        codeSnippetText.textContent =
            `
chmod 600 ${newMachineResponse["machine_name"]}_ssh_key
ssh -p ${newMachineResponse["remote_port"]} -i ${newMachineResponse["machine_name"]}_ssh_key root@${newMachineResponse["remote_ip"]}
            `;
    }
}
