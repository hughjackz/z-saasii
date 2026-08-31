// exi_bridge.cpp — C++ bridge for the C EXI library (clib/exi_lib.h).
// The library header uses C++ syntax (extern "C"), so we compile this file
// as C++ with g++ (auto-detected by CGO from .cpp extension) and expose
// plain C-linkage wrappers for Go to call.
//
// The wrappers are thin; they exist solely to cross the C/C++ boundary.

#include "../../../clib/exi_lib.h"

extern "C" {

int cgo_SA_Set_PKCS8_PriKey(int keyType, unsigned char *keyBuf, int keyLen) {
    return SA_Set_PKCS8_PriKey(keyType, keyBuf, keyLen);
}

int cgo_SA_Set_DER_Cert(int certType, unsigned char *certBuf, int certLen) {
    return SA_Set_DER_Cert(certType, certBuf, certLen);
}

int cgo_SA_Start_Decode_ExiRequest(unsigned char *exirequest_Buf, int exirequest_Len) {
    return SA_Start_Decode_ExiRequest(exirequest_Buf, exirequest_Len);
}

int cgo_SA_Get_Encode_ExiResponse(int *responseResult, unsigned char *exiresponse_Buf, int *exiresponse_Len) {
    return SA_Get_Encode_ExiResponse(responseResult, exiresponse_Buf, exiresponse_Len);
}

void cgo_SA_EXI_Process() {
    SA_EXI_Process();
}

} // extern "C"
