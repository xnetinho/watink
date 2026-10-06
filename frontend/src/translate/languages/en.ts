const messages = {
  en: {
    translations: {
      calls: {
        title: "Calls",
        menu: "Calls",
        unknownContact: "Unnamed contact",
        incoming: {
          title: "Incoming voice call",
          connection: "Connection",
          accept: "Answer",
          reject: "Decline",
        },
        active: {
          calling: "Calling…",
          connecting: "Connecting…",
          inCall: "In call",
          ended: "Call ended",
          hangUp: "Hang up",
          close: "Close",
          mute: "Mute microphone",
          unmute: "Unmute microphone",
          muted: "Microphone muted",
          record: "Record",
          stopRecord: "Stop recording",
          recording: "Recording",
          micDenied: "The browser denied microphone access. Allow it and call again.",
          micUnavailable: "No microphone available.",
          unsupported: "This browser does not support voice calls.",
          socketLost: "The audio connection dropped.",
        },
        pause: {
          pause: "Pause calls",
          resume: "Resume calls",
          paused: "Calls paused in this browser",
        },
        place: {
          button: "Call",
          disconnected: "The connection must be connected to place a call.",
          proxy: "Connections with a proxy cannot place voice calls.",
          notIndividual: "Calls are only available in one-to-one chats.",
          busy: "You are already in a call.",
        },
        quality: {
          title: "Call quality",
          good: "Good",
          fair: "Fair",
          poor: "Poor",
          estimated: "estimated",
          index: "Quality index",
          indexNote: "Index estimated from network measurements; it is not the quality perceived by the contact.",
          rtt: "Latency to relay",
          loss: "Received loss",
          lossNote: "Measured loss is for the audio you receive. Loss on the audio sent to the contact is not measured.",
          jitter: "Jitter",
          bitrateTx: "Send rate",
          bitrateRx: "Receive rate",
          levelTx: "Your microphone level",
          levelRx: "Contact level",
          noPeerAudio: "The contact stopped sending audio.",
          degraded: "Call quality is low.",
          notMeasured: "not measured",
        },
        message: {
          received: "Voice call received",
          made: "Voice call placed",
          missed: "Missed voice call",
          rejected: "Voice call declined",
          interrupted: "Voice call interrupted",
          duration: "Duration",
          handledBy: "Answered by",
          result: "Result",
          listen: "Listen to recording",
          recordingFailed: "Recording failed",
          recordingDeleted: "Recording deleted",
        },
        history: {
          title: "Call history",
          empty: "No calls recorded.",
          direction: "Direction",
          directionIncoming: "Incoming",
          directionOutgoing: "Outgoing",
          contact: "Contact",
          operator: "Operator",
          status: "Status",
          startedAt: "Started",
          duration: "Duration",
          quality: "Quality",
          recording: "Recording",
          filterStatus: "Status",
          filterDirection: "Direction",
          all: "All",
          listen: "Listen",
          delete: "Delete recording",
          deleteConfirmTitle: "Delete recording?",
          deleteConfirmText: "The file will be permanently removed. The deletion is logged.",
          deleted: "Recording deleted.",
          loadError: "Could not load the history.",
        },
        status: {
          ringing: "Ringing",
          active: "In progress",
          ended: "Answered",
          missed: "Missed",
          rejected: "Declined",
          failed: "Failed",
          interrupted: "Interrupted",
        },
        endReason: {
          user_ended: "Ended",
          declined: "Declined",
          timeout: "Not answered",
          busy: "Busy",
          cancelled: "Cancelled",
          failed: "Failure",
          no_operator: "No operator available",
          proxy_blocked: "Connection with proxy",
          unsupported_type: "Unsupported type",
          accepted_elsewhere: "Answered on another device",
          interrupted: "Interrupted",
        },
        alreadyAnswered: "This call was already answered by someone else.",
        settings: {
          title: "Calls",
          description: "Recording of WhatsApp voice calls.",
          mode: "Call recording",
          modeOff: "Off",
          modeOptional: "Optional (the operator decides on each call)",
          modeAuto: "Automatic (every call)",
          unavailable: "This installation has no object storage (S3) configured. Recording is not available.",
          ackTitle: "Statement of responsibility",
          ackText: "Recording phone conversations requires a legal basis and, in many cases, the consent of the people speaking. By turning recording on, your company declares it is solely responsible for informing the parties and for complying with applicable law (including data protection law). Watink does not play an audible notice to the contact.",
          ackCheckbox: "I have read and accept the statement of responsibility",
          ackBy: "Accepted by",
          ackAt: "on",
          save: "Save",
          saved: "Settings saved.",
          ackRequired: "You must accept the statement to turn recording on.",
        },
      },
      common: {
        rowsPerPage: "Rows per page",
        loading: "Loading...",
      },
      role: {
        title: "Roles",
        buttons: {
          add: "Add Role",
        },
        table: {
          name: "Name",
          description: "Description",
          actions: "Actions",
        },
        form: {
          name: "Name",
          description: "Description",
        },
        formTitle: {
          add: "Add Role",
          edit: "Edit Role",
        },
        success: "Role saved successfully!",
        toasts: {
          deleted: "Role deleted successfully.",
        },
        confirmationModal: {
          deleteTitle: "Delete Role",
          deleteMessage: "Are you sure you want to delete this role? This action cannot be undone.",
        },
        permissions: {
          available: "Available Permissions",
          assigned: "Assigned Permissions",
          noPermissions: "No permissions found",
        },
      },
      signup: {
        title: "Sign up",
        toasts: {
          success: "User created successfully! Please login!",
          fail: "Error creating user. Check the reported data.",
        },
        form: {
          name: "Name",
          email: "Email",
          password: "Password",
        },
        buttons: {
          submit: "Register",
          login: "Already have an account? Log in!",
        },
      },
      login: {
        title: "Login",
        form: {
          email: "Email",
          password: "Password",
          rememberMe: "Remember me",
          passwordVisibility: "Toggle password visibility",
        },
        buttons: {
          submit: "Enter",
          register: "Don't have an account? Register!",
        },
      },
      auth: {
        toasts: {
          success: "Login successfully!",
        },
      },
      dashboard: {
        charts: {
          perDay: {
            title: "Tickets today: ",
          },
        },
        messages: {
          inAttendance: {
            title: "In Service"
          },
          waiting: {
            title: "Waiting"
          },
          closed: {
            title: "Closed"
          }
        }
      },
      connections: {
        title: "Connections",
        toasts: {
          deleted: "WhatsApp connection deleted sucessfully!",
          disconnected: "Session disconnected!",
          keepAliveUpdated: "Auto-reconnect updated.",
        },
        confirmationModal: {
          deleteTitle: "Delete",
          deleteMessage: "Are you sure? It cannot be reverted.",
          disconnectTitle: "Disconnect",
          disconnectMessage: "Are you sure? You'll need to read QR Code again.",
        },
        buttons: {
          add: "Add WhatsApp",
          disconnect: "Disconnect",
          tryAgain: "Try Again",
          qrcode: "QR CODE",
          newQr: "New QR CODE",
          connecting: "Connectiing",
          restart: "Restart Connection",
        },
        toolTips: {
          disconnected: {
            title: "Failed to start WhatsApp session",
            content:
              "Make sure your cell phone is connected to the internet and try again, or request a new QR Code",
          },
          qrcode: {
            title: "Waiting for QR Code read",
            content:
              "Click on 'QR CODE' button and read the QR Code with your cell phone to start session",
          },
          connected: {
            title: "Connection established",
          },
          timeout: {
            title: "Connection with cell phone has been lost",
            content:
              "Make sure your cell phone is connected to the internet and WhatsApp is open, or click on 'Disconnect' button to get a new QRcode",
          },
        },
        table: {
          name: "Name",
          status: "Status",
          lastUpdate: "Last Update",
          default: "Default",
          actions: "Actions",
          session: "Session",
        },
      },
      whatsappModal: {
        deleteTitle: "Delete Connection",
        deleteMessage: "Are you sure you want to delete this connection?",
        title: {
          add: "Add WhatsApp",
          edit: "Edit WhatsApp",
        },
        form: {
          name: "Name",
          default: "Default",
        },
        buttons: {
          okAdd: "Add",
          okEdit: "Save",
          cancel: "Cancel",
        },
        success: "WhatsApp saved successfully.",
      },
      qrCode: {
        message: "Read QrCode to start the session",
      },
      contacts: {
        title: "Contacts",
        toasts: {
          deleted: "Contact deleted sucessfully!",
          bulkDeleted: "{{count}} contacts deleted successfully!",
          allDeleted: "All contacts were deleted successfully!",
        },
        searchPlaceholder: "Search ...",
        confirmationModal: {
          deleteTitle: "Delete",
          importTitlte: "Import contacts",
          deleteMessage:
            "Are you sure you want to delete this contact? All related tickets will be lost.",
          importMessage: "Do you want to import all contacts from the phone?",
          bulkDeleteTitle: "Delete selected contacts",
          bulkDeleteMessage:
            "Are you sure you want to delete {{count}} selected contact(s)? This action cannot be undone.",
          deleteAllTitle: "Delete all contacts",
          deleteAllMessage:
            "Warning: this will permanently delete ALL contacts of this tenant, including linked ticket history. This action cannot be undone.",
        },
        buttons: {
          import: "Import Contacts",
          add: "Add Contact",
          deleteSelected: "Delete Selected",
          deselectAll: "Deselect",
          deleteAll: "Delete All Contacts",
        },
        selection: {
          selectedCount: "{{count}} selected",
        },
        table: {
          name: "Name",
          whatsapp: "WhatsApp",
          email: "Email",
          actions: "Actions",
          number: "Phone",
          empty: "No contacts found.",
        },
      },
      contactModal: {
        title: {
          add: "Add contact",
          edit: "Edit contact",
        },
        form: {
          mainInfo: "Contact details",
          extraInfo: "Additional information",
          name: "Name",
          number: "Whatsapp number",
          email: "Email",
          extraName: "Field name",
          extraValue: "Value",
        },
        buttons: {
          addExtraInfo: "Add information",
          okAdd: "Add",
          okEdit: "Save",
          cancel: "Cancel",
        },
        success: "Contact saved successfully.",
      },
      quickAnswersModal: {
        title: {
          add: "Add Quick Reply",
          edit: "Edit Quick Answer",
        },
        form: {
          shortcut: "Shortcut",
          message: "Quick Reply",
        },
        buttons: {
          okAdd: "Add",
          okEdit: "Save",
          cancel: "Cancel",
        },
        success: "Quick Reply saved successfully.",
      },
      queueModal: {
        title: {
          add: "Add queue",
          edit: "Edit queue",
        },
        form: {
          name: "Name",
          color: "Color",
          greetingMessage: "Greeting Message",
          connection: "Connections",
          selectConnection: "Select connections",
          hierarchy: "Hierarchy",
          parentQueue: "Parent queue",
          none: "None",
          distributionSection: "Ticket distribution",
          distributionStrategy: "Distribution strategy",
          prioritizeWallet: "Prioritize wallet",
          prioritizeWalletHelp: "Tickets are preferably assigned to the contact wallet owner",
          prioritizeWalletTooltip: "When enabled, the system checks whether the contact has an assigned salesperson/agent in their wallet. If that agent is online and belongs to this queue, the ticket is automatically assigned to them.",
        },
        strategies: {
          manual: "Manual (Pickup)",
          manualDescription: "Agents choose which tickets to handle",
          roundRobin: "Automated (Round Robin)",
          roundRobinDescription: "Distributes tickets evenly among available agents",
          balanced: "Automated (Balanced)",
          balancedDescription: "Prioritizes agents with fewer open tickets",
        },
        buttons: {
          okAdd: "Add",
          okEdit: "Save",
          cancel: "Cancel",
        },
      },
      userModal: {
        title: {
          add: "Add user",
          edit: "Edit user",
        },
        form: {
          name: "Name",
          email: "Email",
          password: "Password",
          profile: "Profile",
          whatsapp: "Default Connection",
        },
        buttons: {
          okAdd: "Add",
          okEdit: "Save",
          cancel: "Cancel",
        },
        success: "User saved successfully.",
      },
      chat: {
        noTicketMessage: "Select a ticket to start chatting.",
      },
      ticketsManager: {
        buttons: {
          newTicket: "New",
        },
      },
      ticketsQueueSelect: {
        placeholder: "Queues",
      },
      tickets: {
        toasts: {
          deleted: "The ticket you were on has been deleted.",
        },
        notification: {
          message: "Message from",
        },
        tabs: {
          open: { title: "Inbox" },
          closed: { title: "Resolved" },
          search: { title: "Search" },
        },
        search: {
          placeholder: "Search tickets and messages.",
        },
        buttons: {
          showAll: "All",
        },
      },
      transferTicketModal: {
        title: "Transfer Ticket",
        fieldLabel: "Type to search for users",
        fieldQueueLabel: "Transfer to queue",
        fieldConnectionLabel: "Transfer to connection",
        fieldQueuePlaceholder: "Please select a queue",
        fieldConnectionPlaceholder: "Please select a connection",
        noOptions: "No user found with this name",
        buttons: {
          ok: "Transfer",
          cancel: "Cancel",
        },
      },
      ticketsList: {
        pendingHeader: "Queue",
        assignedHeader: "Working on",
        noTicketsTitle: "Nothing here!",
        noTicketsMessage: "No tickets found with this status or search term.",
        connectionTitle: "Connection that is currently being used.",
        buttons: {
          accept: "Accept",
        },
      },
      newTicketModal: {
        title: "Create Ticket",
        fieldLabel: "Type to search for a contact",
        add: "Add",
        buttons: {
          ok: "Save",
          cancel: "Cancel",
        },
      },
      mainDrawer: {
        listItems: {
          dashboard: "Dashboard",
          assistants: "AI Assistants",
          pipelines: "Pipelines",
          tickets: "Tickets",
          myActivities: "My Activities",
          activities: "Activities",
          contacts: "Contacts",
          quickAnswers: "Quick Answers",
          flowBuilder: "Flow Builder",
          clients: "Clients",
          helpdesk: "Helpdesk",
          whatsappGroupsHub: "WhatsApp Groups",
          calls: "Calls",
          administration: "Administration",
          tags: "Tags",
          groups: "Groups",
          connections: "Connections",
          users: "Users",
          queues: "Queues",
          knowledgeBase: "Knowledge Base",
          settings: "Settings",
          swagger: "Swagger",
        },
        appBar: {
          user: {
            profile: "Profile",
            logout: "Logout",
          },
        },
      },
      notifications: {
        noTickets: "No notifications.",
      },
      queues: {
        title: "Queues",
        table: {
          name: "Name",
          color: "Color",
          greeting: "Greeting message",
          connections: "Connections",
          actions: "Actions",
        },
        buttons: {
          add: "Add queue",
        },
        confirmationModal: {
          deleteTitle: "Delete",
          deleteMessage:
            "Are you sure? It cannot be reverted! Tickets in this queue will still exist, but will not have any queues assigned.",
        },
      },
      queueSelect: {
        inputLabel: "Queues",
      },
      quickAnswers: {
        title: "Quick Answers",
        table: {
          shortcut: "Shortcut",
          message: "Quick Reply",
          actions: "Actions",
        },
        buttons: {
          add: "Add Quick Reply",
        },
        toasts: {
          deleted: "Quick Reply deleted successfully.",
        },
        searchPlaceholder: "Search...",
        confirmationModal: {
          deleteTitle: "Are you sure you want to delete this Quick Reply: ",
          deleteMessage: "This action cannot be undone.",
        },
      },
      flowBuilder: {
        nodes: {
          quickAnswer: {
            paletteLabel: "Quick Reply",
            title: "Configure Quick Reply",
            fieldLabel: "Quick Reply",
            placeholder: "Select a quick reply",
            empty: "No quick replies registered",
            helper: "Sends a template from the Quick Replies module.",
          },
        },
      },
      users: {
        title: "Users",
        table: {
          name: "Name",
          email: "Email",
          profile: "Profile",
          whatsapp: "Default Connection",
          actions: "Actions",
        },
        buttons: {
          add: "Add user",
        },
        toasts: {
          deleted: "User deleted sucessfully.",
        },
        confirmationModal: {
          deleteTitle: "Delete",
          deleteMessage:
            "All user data will be lost. Users' open tickets will be moved to queue.",
        },
      },
      settings: {
        success: "Settings saved successfully.",
        title: "Settings",
        settings: {
          userCreation: {
            name: "User creation",
            options: {
              enabled: "Enabled",
              disabled: "Disabled",
            },
          },
          language: {
            name: "Language",
            options: {
              pt: "Portuguese",
              en: "English",
              es: "Spanish",
            },
          },
        },
      },
      messagesList: {
        header: {
          assignedTo: "Assigned to:",
          buttons: {
            return: "Return",
            resolve: "Resolve",
            reopen: "Reopen",
            accept: "Accept",
          },
        },
      },
      recoverHistory: {
        button: "History",
        title: "Recover history",
        requested: "History recovery requested. Messages will appear shortly.",
        ranges: {
          oneDay: "Last day",
          twoDays: "Last 2 days",
          oneWeek: "Last week",
          oneMonth: "Last month",
          all: "All available history",
        },
      },
      messagesInput: {
        placeholderOpen: "Type a message or press ''/'' to use the registered quick responses",
        placeholderClosed: "Reopen or accept this ticket to send a message.",
        signMessage: "Sign",
      },
      contactDrawer: {
        header: "Contact details",
        buttons: {
          edit: "Edit contact",
        },
        extraInfo: "Other information",
      },
      ticketOptionsMenu: {
        delete: "Delete",
        transfer: "Transfer",
        confirmationModal: {
          title: "Delete ticket #",
          titleFrom: "from contact ",
          message: "Attention! All ticket's related messages will be lost.",
        },
        buttons: {
          delete: "Delete",
          cancel: "Cancel",
        },
      },
      confirmationModal: {
        buttons: {
          confirm: "Ok",
          cancel: "Cancel",
        },
      },
      messageOptionsMenu: {
        delete: "Delete",
        reply: "Reply",
        confirmationModal: {
          title: "Delete message?",
          message: "This action cannot be reverted.",
        },
      },
      access: {
        title: "Access and Permissions",
        metrics: {
          roles: "Active roles",
          noRole: "Users without role",
          total: "Total users",
        },
        buttons: {
          manageRoles: "Manage Roles",
          manageUsers: "Manage Users",
          managePermissions: "Manage Permissions",
          legacyGroups: "Legacy Groups",
        },
      },
      backendErrors: {
        ERR_NO_OTHER_WHATSAPP:
          "There must be at lest one default WhatsApp connection.",
        ERR_NO_DEF_WAPP_FOUND:
          "No default WhatsApp found. Check connections page.",
        ERR_WAPP_NOT_INITIALIZED:
          "This WhatsApp session is not initialized. Check connections page.",
        ERR_WAPP_CHECK_CONTACT:
          "Could not check WhatsApp contact. Check connections page.",
        ERR_WAPP_INVALID_CONTACT: "This is not a valid whatsapp number.",
        ERR_WAPP_DOWNLOAD_MEDIA:
          "Could not download media from WhatsApp. Check connections page.",
        ERR_INVALID_CREDENTIALS: "Authentication error. Please try again.",
        ERR_SENDING_WAPP_MSG:
          "Error sending WhatsApp message. Check connections page.",
        ERR_DELETE_WAPP_MSG: "Couldn't delete message from WhatsApp.",
        ERR_OTHER_OPEN_TICKET:
          "There's already an open ticket for this contact.",
        ERR_SESSION_EXPIRED: "Session expired. Please login.",
        ERR_USER_CREATION_DISABLED:
          "User creation was disabled by administrator.",
        ERR_NO_PERMISSION: "You don't have permission to access this resource.",
        ERR_DUPLICATED_CONTACT: "A contact with this number already exists.",
        ERR_NO_SETTING_FOUND: "No setting found with this ID.",
        ERR_NO_CONTACT_FOUND: "No contact found with this ID.",
        ERR_NO_TICKET_FOUND: "No ticket found with this ID.",
        ERR_NO_USER_FOUND: "No user found with this ID.",
        ERR_NO_WAPP_FOUND: "No WhatsApp found with this ID.",
        ERR_CREATING_MESSAGE: "Error while creating message on database.",
        ERR_CREATING_TICKET: "Error while creating ticket on database.",
        ERR_FETCH_WAPP_MSG:
          "Error fetching the message in WhtasApp, maybe it is too old.",
        ERR_QUEUE_COLOR_ALREADY_EXISTS:
          "This color is already in use, pick another one.",
        ERR_WAPP_GREETING_REQUIRED:
          "Greeting message is required if there is more than one queue.",
        plan_limit_reached: "Your plan's limit has been reached.",
      },
      planLimitResources: {
        users: "users",
        connections: "connections",
        queues: "queues",
        plugins: "plugins",
      },
      accountSuspended: {
        suspendedTitle: "Account suspended",
        suspendedMessage:
          "Access to this account is temporarily suspended. Contact support to regularize your situation and resume using the platform.",
        canceledTitle: "Account canceled",
        canceledMessage:
          "This account has been canceled. Contact support if you believe this is a mistake.",
        backToLogin: "Back to login",
      },
    },
  },
};

export { messages };
