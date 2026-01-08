export const mockResponses = {
  signin: {
    start: {
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        token:
          "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJDMGFfWk1kNlAwRlkwc25CR1FsY2c5ZjdjQkNzb0NfOU90NFhiTjl1TS1VIiwiZXhwIjoxNzU2ODM5Mjc1fQ.XVzBg_UokcVkgV96o4Oxb6iCLkqK8Tk8Nnkk2Js_Imo",
        assertionRequest: {
          challenge: "C0a_ZMd6P0FY0snBGQlcg9f7cBCsoC_9Ot4XbN9uM-U",
          timeout: 300000,
          rpId: "test.example.com",
          allowCredentials: [],
          userVerification: "required",
        },
      }),
    },
    email: {
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        success: {
          token: "01990bc4-7195-70cb-98f7-5c372f46d51b",
          assertionRequest: {
            challenge: "vYjyXKH2G0mCQf5rLVs06KNrvVcfZWejVmn3L3TAyVc",
            timeout: 300000,
            rpId: "test.example.com",
            allowCredentials: [
              {
                type: "public-key",
                id: "uFI1e3f9tsqco2LXr_vpyIaeO-c",
                transports: ["hybrid", "internal"],
              },
            ],
            userVerification: "required",
          },
        },
      }),
    },
    webauthn: {
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        success: {
          cookie: "cookie",
          redirect: "/",
        },
      }),
    },
  },
  user: {
    get: {
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        id: "u1",
        email: "hello@example.com",
        displayName: "John Doe",
        isAdmin: true,
        credentials: [],
        sessions: [],
        currentSession: null,
        sshKeys: [],
      }),
    },
  },
};
