var BACKEND = "http://" + window.location.host

export async function GetStatus(api, callback) {
    if (api.includes("ncaaf")) {
        console.log(`Calling status /api/${api}`)
    }
    return fetch(`${BACKEND}/api/${api}`,
        {
            method: "GET",
            mode: "cors",
        }
    ).then((resp) => {
        if (resp.ok) {
            return resp.text();
        }
        throw resp
    }
    ).then((status) => {
        if (status === "true") {
            if (api.includes("ncaaf")) {
                console.log(`Status ${api}: ${status}`)
            }
            callback(true)
        } else if (status === "false") {
            if (api.includes("ncaaf")) {
                console.log(`Status ${api}: ${status}`)
            }
            callback(false)
        }
    }
    ).catch(error => {
        console.log(`Error calling /api/${api}: ` + error)
    })

}
export async function CallMatrix(path) {
    console.log(`Calling matrix API ${path}`)
    return fetch(`${BACKEND}/api/${path}`, {
        method: "GET",
        mode: "cors",
    }).then((resp) => {
        console.log(`Response ${path}: ${resp.status}`);
    });
}
export async function MatrixPost(path, body) {
    const req = {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: body,
    }
    console.log(`Matrix POST ${body}`)
    return fetch(`${BACKEND}/api/${path}`, req).then((resp) => {
        if (resp.ok) {
            console.log(`POST ${path}: ${resp.status}`)
            return
        }
        console.log(`POST ${path}: ${resp.status}`)
        throw resp
    }).catch(error => console.log(`POST ERROR: ${error}`))
}
export async function GetVersion(callback) {
    return fetch(`${BACKEND}/api/version`, {
        method: "GET",
        mode: "cors",
    }).then((resp) => {
        if (resp.ok) {
            return resp.text();
        }
        throw resp
    }).then((ver) => {
        callback(ver);
    }).catch(error => console.log(`ERROR /api/version: ${error}`))
}