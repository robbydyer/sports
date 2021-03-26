var BACKEND = "http://" + window.location.host

export async function GetStatus(api, callback) {
    var resp = await fetch(`${BACKEND}/api/${api}`,
        {
            method: "GET",
            mode: "cors",
        }
    );

    const status = await resp.text();

    if (resp.ok) {
        if (status === "true") {
            callback(true)
        } else {
            callback(false)
        }
    }
}
export function CallMatrix(path) {
    console.log(`Calling matrix API ${path}`)
    var resp = fetch(`${BACKEND}/api/${path}`, {
        method: "GET",
        mode: "cors",
    });
    console.log("Response", resp.ok);
}
export async function GetVersion(callback) {
    var resp = await fetch(`${BACKEND}/api/version`, {
        method: "GET",
        mode: "cors",
    });
    const ver = await resp.text();
    callback(ver);
}