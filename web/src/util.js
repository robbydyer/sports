import * as basicboard_pb from './basicboard/basicboard_pb';
import * as sportsmatrix_pb from './sportsmatrix/sportsmatrix_pb';

export var BACKEND = "http://" + window.location.host

export function MatrixPostRet(path, body) {
    const req = {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: body,
    }
    //console.log(`Matrix POST ${BACKEND}/${path} ${body}`)
    return fetch(`${BACKEND}/${path}`, req)
}
export async function GetVersion(callback) {
    return await MatrixPostRet("matrix.v1.Sportsmatrix/Version", '{}').then((resp) => {
        if (resp.ok) {
            return resp.text()
        }
        throw resp
    }).then((data) => {
        var d = JSON.parse(data);
        callback(d.version);
    }).catch(err => {
        console.log("failed to get version", err);
    }
    );
}

export function JSONToStatus(jsonDat) {
    var d = JSON.parse(jsonDat);
    var dat = d.status;
    var status = new basicboard_pb.Status();
    status.setEnabled(dat.enabled);

    return status;
}

export async function JumpToBoard(board) {
    var req = new sportsmatrix_pb.JumpReq();
    req.setBoard(board);
    var r = JSON.stringify(req.toObject());
    console.log("Board Jump", "matrix.v1.Sportsmatrix/Jump", r);
    await MatrixPostRet("matrix.v1.Sportsmatrix/Jump", r);
}