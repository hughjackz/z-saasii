
#ifndef _SA_OCPP_LIB_H
#define _SA_OCPP_LIB_H


enum SA_PRIKEY_TYPE
{
	PriKey_CPS_Leaf,
	PriKey_Contarct_Leaf_Install,
	PriKey_Contarct_Leaf_Update,
};



enum SA_CERT_TYPE
{
	CERT_MO_ROOT,
		
	CERT_OEM_ROOT,
	CERT_OEM_SubCA1,
	CERT_OEM_SubCA2,
	
	CERT_Contarct_SubCA1_Install,
	CERT_Contarct_SubCA2_Install,
	CERT_Contarct_Leaf_Install,

	CERT_Contarct_SubCA1_Update,
	CERT_Contarct_SubCA2_Update,
	CERT_Contarct_Leaf_Update,

	CERT_CPS_SuBCA1,
	CERT_CPS_SuBCA2,
	CERT_CPS_Leaf,
	
};

extern "C"{

/*
10ms调用一次，exiresponse和exirequest计算会阻塞，建议独立线程
*/
void SA_EXI_Process();


/*
1.调用SA_EXI_Process前，必须调用
2.设置3种私钥，必须全部设置
	PriKey_CPS_Leaf
	PriKey_Contarct_Leaf_Install
	PriKey_Contarct_Leaf_Update
3.私钥格式必须为pkcs#8格式的DER格式，一般长度<256字节
4.返回值：
	0：设置成功
	1：keyBuf为空
	2：keyLen长度越界
	3：未知keyType私钥定义
*/
int SA_Set_PKCS8_PriKey(int keyType,unsigned char *keyBuf,int keyLen);


/*
1.调用SA_EXI_Process前，必须调用
2.设置13种证书，必须全部设置
	CERT_MO_ROOT		
	CERT_OEM_ROOT
	CERT_OEM_SubCA1
	CERT_OEM_SubCA2
	CERT_Contarct_SubCA1_Install
	CERT_Contarct_SubCA2_Install
	CERT_Contarct_Leaf_Install
	CERT_Contarct_SubCA1_Update
	CERT_Contarct_SubCA2_Update
	CERT_Contarct_Leaf_Update
	CERT_CPS_SuBCA1
	CERT_CPS_SuBCA2
	CERT_CPS_Leaf
3.证书格式必须为DER格式，一般长度<800字节
4.返回值：
	0：设置成功
	1：certBuf为空
	2：certLen长度越界
	3：未知certType证书定义
*/
int SA_Set_DER_Cert(int certType,unsigned char *certBuf,int certLen);

/*
1.调用SA_EXI_Process前，必须调用
2.平台收到EVSE的exirequest后，调用本函数开始执行解码
3.exirequest格式需对ocpp原格式进行base64解码成二进制传入exirequest_Buf
4.返回值：
	0：设置成功
	1：exirequest_Buf为空
	2：exirequest_Len长度越界
*/
int SA_Start_Decode_ExiRequest(unsigned char *exirequest_Buf,int exirequest_Len);


/*
1.调用SA_EXI_Process前，必须调用
2.每10ms获取解码结果，并获取exiresponse编码，调用本函数获取结果
3.平台拿到exiresponse后，需进行base64编码，exiresponse_Buf是二进制格式
4.返回值：
	0：设置成功
	1：exirequest_Buf为空
	2：exirequest_Len长度越界
5、responseResult结果
	1：response获取成功
	0：response还在编解码过程中
*/
int SA_Get_Encode_ExiResponse(int *responseResult,unsigned char *exiresponse_Buf,int *exiresponse_Len);

}
#endif

