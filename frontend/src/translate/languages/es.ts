const messages = {
  es: {
    translations: {
      calls: {
        title: "Llamadas",
        menu: "Llamadas",
        unknownContact: "Contacto sin nombre",
        incoming: {
          title: "Llamada de voz entrante",
          connection: "Conexión",
          videoTitle: "Videollamada entrante",
          videoUnsupported: "Este navegador no muestra video: la llamada se atenderá solo con audio. Use Chrome, Edge o Brave para ver la imagen.",
          accept: "Contestar",
          reject: "Rechazar",
        },
        active: {
          calling: "Llamando…",
          connecting: "Conectando…",
          inCall: "En llamada",
          ended: "Llamada finalizada",
          endedBy: {
            user_ended: "El contacto colgó",
            declined: "El contacto rechazó la llamada",
            timeout: "El contacto no respondió",
            busy: "El contacto está ocupado",
            cancelled: "Llamada cancelada",
            failed: "La llamada falló",
            accepted_elsewhere: "Atendida en otro dispositivo",
            media_timeout: "Sin audio: la conexión de medios no se abrió",
            local: "Terminaste la llamada",
          },
          hangUp: "Finalizar",
          videoOf: "Video del contacto",
          videoPaused: "El contacto desactivó la cámara",
          videoUnsupported: "Para ver el video, use Chrome, Edge o Brave. La llamada continúa solo con audio.",
          cameraOn: "Encender cámara",
          cameraOff: "Apagar cámara",
          cameraDenied: "El navegador bloqueó la cámara. Permita el acceso a la cámara desde el candado de la barra de direcciones e inténtelo de nuevo.",
          cameraUnavailable: "No se pudo usar la cámara: no existe o está en uso por otro programa.",
          cameraEncoder: "La cámara se detuvo porque el navegador no pudo codificar el video.",
          cameraUnsupported: "Para enviar su cámara, use Chrome, Edge o Brave.",
          close: "Cerrar",
          mute: "Silenciar micrófono",
          unmute: "Activar micrófono",
          muted: "Micrófono silenciado",
          record: "Grabar",
          stopRecord: "Detener grabación",
          recording: "Grabando",
          micDenied: "El navegador denegó el acceso al micrófono. Permítalo y llame de nuevo.",
          micUnavailable: "No hay micrófono disponible.",
          unsupported: "Este navegador no admite llamadas de voz.",
          socketLost: "Se cayó la conexión de audio.",
          audioUnavailable: "El canal de audio del servidor no está disponible. Avise al administrador (ENGINE_HOST del business).",
          riskNotice: "Las llamadas por un canal no oficial aumentan el riesgo de bloqueo del número. Ese riesgo es de la empresa.",
        },
        pause: {
          pause: "Pausar llamadas",
          resume: "Reanudar llamadas",
          paused: "Llamadas en pausa en este navegador",
        },
        place: {
          button: "Llamar",
          disconnected: "La conexión debe estar conectada para llamar.",
          proxy: "Las conexiones con proxy no pueden hacer llamadas de voz.",
          notIndividual: "Solo se puede llamar en conversaciones individuales.",
          busy: "Ya está en una llamada.",
        },
        quality: {
          title: "Calidad de la llamada",
          good: "Buena",
          fair: "Regular",
          poor: "Mala",
          estimated: "estimado",
          index: "Índice de calidad",
          indexNote: "Índice estimado a partir de mediciones de red; no es la calidad percibida por el contacto.",
          rtt: "Latencia al relay",
          loss: "Pérdida recibida",
          lossNote: "La pérdida medida es la del audio que usted recibe. La pérdida en el envío al contacto no se mide.",
          jitter: "Jitter",
          bitrateTx: "Tasa de envío",
          bitrateRx: "Tasa de recepción",
          levelTx: "Nivel de su micrófono",
          levelRx: "Nivel del contacto",
          noPeerAudio: "El contacto dejó de enviar audio.",
          degraded: "La calidad de la llamada es baja.",
          notMeasured: "no medido",
        },
        message: {
          received: "Llamada de voz recibida",
          made: "Llamada de voz realizada",
          missed: "Llamada de voz perdida",
          rejected: "Llamada de voz rechazada",
          interrupted: "Llamada de voz interrumpida",
          videoReceived: "Videollamada recibida",
          videoMade: "Videollamada realizada",
          videoMissed: "Videollamada perdida",
          videoRejected: "Videollamada rechazada",
          videoInterrupted: "Videollamada interrumpida",
          duration: "Duración",
          handledBy: "Atendida por",
          result: "Resultado",
          listen: "Escuchar grabación",
          recordingFailed: "La grabación falló",
          recordingDeleted: "Grabación eliminada",
        },
        history: {
          title: "Historial de llamadas",
          empty: "No hay llamadas registradas.",
          direction: "Dirección",
          directionIncoming: "Recibida",
          directionOutgoing: "Realizada",
          contact: "Contacto",
          operator: "Operador",
          status: "Estado",
          startedAt: "Inicio",
          duration: "Duración",
          quality: "Calidad",
          recording: "Grabación",
          filterStatus: "Estado",
          filterDirection: "Dirección",
          all: "Todas",
          listen: "Escuchar",
          delete: "Eliminar grabación",
          deleteConfirmTitle: "¿Eliminar grabación?",
          deleteConfirmText: "El archivo se eliminará definitivamente. La eliminación queda registrada.",
          deleted: "Grabación eliminada.",
          loadError: "No se pudo cargar el historial.",
        },
        status: {
          ringing: "Sonando",
          active: "En curso",
          ended: "Atendida",
          missed: "Perdida",
          rejected: "Rechazada",
          failed: "Falló",
          interrupted: "Interrumpida",
        },
        endReason: {
          user_ended: "Finalizada",
          declined: "Rechazada",
          timeout: "Sin respuesta",
          busy: "Ocupado",
          cancelled: "Cancelada",
          failed: "Falla",
          no_operator: "Sin operador disponible",
          proxy_blocked: "Conexión con proxy",
          unsupported_type: "Tipo no admitido",
          accepted_elsewhere: "Atendida en otro dispositivo",
          media_timeout: "Sin audio: la conexión de medios no se abrió (verifique la salida UDP del servidor)",
          interrupted: "Interrumpida",
        },
        alreadyAnswered: "Esta llamada ya fue atendida por otra persona.",
        settings: {
          title: "Llamadas",
          description: "Grabación de las llamadas de voz de WhatsApp.",
          mode: "Grabación de llamadas",
          modeOff: "Desactivada",
          modeOptional: "Opcional (el operador decide en cada llamada)",
          modeAuto: "Automática (todas las llamadas)",
          unavailable: "Esta instalación no tiene almacenamiento de objetos (S3) configurado. La grabación no está disponible.",
          ackTitle: "Declaración de responsabilidad",
          ackText: "Grabar conversaciones telefónicas requiere una base legal y, en muchos casos, el consentimiento de quienes hablan. Al activar la grabación, su empresa declara ser la única responsable de informar a los interlocutores y de cumplir la legislación aplicable (incluida la de protección de datos). Watink no emite un aviso sonoro al contacto.",
          ackCheckbox: "He leído y acepto la declaración de responsabilidad",
          ackBy: "Aceptado por",
          ackAt: "el",
          save: "Guardar",
          saved: "Configuración guardada.",
          ackRequired: "Debe aceptar la declaración para activar la grabación.",
        },
      },
      common: {
        rowsPerPage: "Filas por página",
        loading: "Cargando...",
      },
      role: {
        title: "Roles",
        buttons: {
          add: "Agregar rol",
        },
        table: {
          name: "Nombre",
          description: "Descripción",
          actions: "Acciones",
        },
        form: {
          name: "Nombre",
          description: "Descripción",
        },
        formTitle: {
          add: "Agregar rol",
          edit: "Editar rol",
        },
        success: "Rol guardado correctamente!",
        toasts: {
          deleted: "Rol eliminado correctamente.",
        },
        confirmationModal: {
          deleteTitle: "Eliminar rol",
          deleteMessage: "¿Estás seguro de que deseas eliminar este rol? Esta acción no se puede deshacer.",
        },
        permissions: {
          available: "Permisos disponibles",
          assigned: "Permisos asignados",
          noPermissions: "No se encontraron permisos",
        },
      },
      signup: {
        title: "Registro",
        toasts: {
          success:
            "¡El usuario ha sido creado satisfactoriamente! ¡Ahora inicia sesión!",
          fail: "Error creando el usuario. Verifica la data reportada.",
        },
        form: {
          name: "Nombre",
          email: "Correo Electrónico",
          password: "Contraseña",
        },
        buttons: {
          submit: "Regístrate",
          login: "¿Ya tienes una cuenta? ¡Inicia sesión!",
        },
      },
      login: {
        title: "Inicio de Sesión",
        form: {
          email: "Correo Electrónico",
          password: "Contraseña",
          rememberMe: "Recuérdame",
          passwordVisibility: "Alternar visibilidad de contraseña",
        },
        buttons: {
          submit: "Ingresa",
          register: "¿No tienes cuenta? ¡Regístrate!",
        },
      },
      auth: {
        toasts: {
          success: "¡Inicio de sesión exitoso!",
        },
      },
      dashboard: {
        charts: {
          perDay: {
            title: "Tickets hoy: ",
          },
        },
        messages: {
          inAttendance: {
            title: "En servicio"
          },
          waiting: {
            title: "Esperando"
          },
          closed: {
            title: "Finalizado"
          }
        }
      },
      connections: {
        title: "Conexiones",
        invalidPhone: "Número de teléfono no válido",
        presencePolicy: {
          title: "Presencia del número",
          fieldLabel: "Cuándo el número está disponible (en línea)",
          ariaLabel: "Política de presencia",
          idleLabel: "Volver a lo normal tras (minutos sin operador)",
          updated: "Política de presencia actualizada.",
          never: {
            label: "Nunca disponible",
            summary:
              "El teléfono sigue recibiendo notificaciones. El contacto no ve este número en línea. El estado en línea y la última vez de los contactos casi no aparecen en Watink.",
          },
          on_demand: {
            label: "Disponible durante la atención",
            summary:
              "El número queda disponible solo mientras un operador tiene abierta una conversación de esta conexión, y vuelve a lo normal tras el tiempo de inactividad. El teléfono solo pierde las notificaciones en ese período.",
          },
          always: {
            label: "Siempre disponible",
            summary:
              "El número está siempre disponible, incluso al reconectar. El estado en línea y la última vez de los contactos siempre aparecen, pero el teléfono no recibe notificaciones mientras la conexión esté activa.",
          },
        },
        toasts: {
          deleted:
            "¡La conexión de WhatsApp ha sido borrada satisfactoriamente!",
          disconnected: "¡Sesión desconectada!",
          keepAliveUpdated: "Reconexión automática actualizada.",
        },
        confirmationModal: {
          deleteTitle: "Borrar",
          deleteMessage: "¿Estás seguro? Este proceso no puede ser revertido.",
          disconnectTitle: "Desconectar",
          disconnectMessage: "Estás seguro? Deberá volver a leer el código QR",
        },
        buttons: {
          add: "Agrega WhatsApp",
          disconnect: "Desconectar",
          tryAgain: "Inténtalo de nuevo",
          qrcode: "QR CODE",
          newQr: "Nuevo QR CODE",
          connecting: "Conectando",
          restart: "Reiniciar Conexión",
        },
        toolTips: {
          disconnected: {
            title: "No se pudo iniciar la sesión de WhatsApp",
            content:
              "Asegúrese de que su teléfono celular esté conectado a Internet y vuelva a intentarlo o solicite un nuevo código QR",
          },
          qrcode: {
            title: "Esperando la lectura del código QR",
            content:
              "Haga clic en el botón 'CÓDIGO QR' y lea el Código QR con su teléfono celular para iniciar la sesión",
          },
          connected: {
            title: "Conexión establecida",
          },
          timeout: {
            title: "Se perdió la conexión con el teléfono celular",
            content:
              "Asegúrese de que su teléfono celular esté conectado a Internet y que WhatsApp esté abierto, o haga clic en el botón 'Desconectar' para obtener un nuevo código QR",
          },
        },
        table: {
          name: "Nombre",
          status: "Estado",
          lastUpdate: "Última Actualización",
          default: "Por Defecto",
          actions: "Acciones",
          session: "Sesión",
        },
      },
      whatsappModal: {
        deleteTitle: "Eliminar conexión",
        deleteMessage: "¿Estás seguro de que deseas eliminar esta conexión?",
        title: {
          add: "Agrega WhatsApp",
          edit: "Edita WhatsApp",
        },
        form: {
          name: "Nombre",
          default: "Por Defecto",
        },
        buttons: {
          okAdd: "Agregar",
          okEdit: "Guardar",
          cancel: "Cancelar",
        },
        success: "WhatsApp guardado satisfactoriamente.",
      },
      qrCode: {
        message: "Lée el código QR para empezar la sesión.",
      },
      contacts: {
        title: "Contactos",
        toasts: {
          deleted: "¡Contacto borrado satisfactoriamente!",
          bulkDeleted: "¡{{count}} contactos borrados satisfactoriamente!",
          allDeleted: "¡Todos los contactos fueron borrados satisfactoriamente!",
        },
        searchPlaceholder: "Buscar...",
        confirmationModal: {
          deleteTitle: "Borrar",
          importTitlte: "Importar contactos",
          deleteMessage:
            "¿Estás seguro que deseas borrar este contacto? Todos los tickets relacionados se perderán.",
          importMessage:
            "¿Quieres importar todos los contactos desde tu teléfono?",
          bulkDeleteTitle: "Borrar contactos seleccionados",
          bulkDeleteMessage:
            "¿Estás seguro que deseas borrar {{count}} contacto(s) seleccionado(s)? Esta acción no se puede deshacer.",
          deleteAllTitle: "Borrar todos los contactos",
          deleteAllMessage:
            "Atención: esto borrará TODOS los contactos de este tenant permanentemente, incluyendo el historial de tickets vinculado. Esta acción no se puede deshacer.",
        },
        buttons: {
          import: "Importar Contactos",
          add: "Agregar Contacto",
          deleteSelected: "Borrar Seleccionados",
          deselectAll: "Desmarcar",
          deleteAll: "Borrar Todos los Contactos",
        },
        selection: {
          selectedCount: "{{count}} seleccionado(s)",
        },
        table: {
          name: "Nombre",
          whatsapp: "WhatsApp",
          email: "Correo Electrónico",
          actions: "Acciones",
          number: "Teléfono",
          empty: "Ningún contacto encontrado.",
        },
      },
      contactModal: {
        title: {
          add: "Agregar contacto",
          edit: "Editar contacto",
        },
        form: {
          mainInfo: "Detalles del contacto",
          extraInfo: "Información adicional",
          name: "Nombre",
          number: "Número de Whatsapp",
          email: "Correo Electrónico",
          extraName: "Nombre del Campo",
          extraValue: "Valor",
        },
        buttons: {
          addExtraInfo: "Agregar información",
          okAdd: "Agregar",
          okEdit: "Guardar",
          cancel: "Cancelar",
        },
        success: "Contacto guardado satisfactoriamente.",
      },
      quickAnswersModal: {
        title: {
          add: "Agregar respuesta rápida",
          edit: "Editar respuesta rápida",
        },
        form: {
          shortcut: "Atajo",
          message: "Respuesta rápida",
        },
        buttons: {
          okAdd: "Agregar",
          okEdit: "Guardar",
          cancel: "Cancelar",
        },
        success: "Respuesta rápida guardada correctamente.",
      },
      queueModal: {
        title: {
          add: "Agregar cola",
          edit: "Editar cola",
        },
        form: {
          name: "Nombre",
          color: "Color",
          greetingMessage: "Mensaje de saludo",
          connection: "Conexiones",
          selectConnection: "Seleccione las conexiones",
          hierarchy: "Jerarquía",
          parentQueue: "Cola padre",
          none: "Ninguna",
          distributionSection: "Distribución de tickets",
          distributionStrategy: "Estrategia de distribución",
          prioritizeWallet: "Priorizar cartera",
          prioritizeWalletHelp: "Los tickets se asignan preferentemente al dueño de la cartera del contacto",
          prioritizeWalletTooltip: "Cuando está activo, el sistema verifica si el contacto tiene un vendedor/agente responsable asignado a su cartera. Si ese agente está en línea y pertenece a esta cola, el ticket se asigna automáticamente a él.",
        },
        strategies: {
          manual: "Manual",
          manualDescription: "Los agentes eligen qué tickets atender",
          roundRobin: "Automático circular",
          roundRobinDescription: "Distribuye tickets equitativamente entre agentes disponibles",
          balanced: "Automático balanceado",
          balancedDescription: "Prioriza agentes con menos tickets abiertos",
        },
        buttons: {
          okAdd: "Añadir",
          okEdit: "Ahorrar",
          cancel: "Cancelar",
        },
      },
      userModal: {
        title: {
          add: "Agregar usuario",
          edit: "Editar usuario",
        },
        form: {
          name: "Nombre",
          email: "Correo Electrónico",
          password: "Contraseña",
          profile: "Perfil",
          whatsapp: "Conexión estándar",
        },
        buttons: {
          okAdd: "Agregar",
          okEdit: "Guardar",
          cancel: "Cancelar",
        },
        success: "Usuario guardado satisfactoriamente.",
      },
      chat: {
        unsupported: {
          title: "Mensaje no soportado",
          hint: "Watink aún no puede mostrar este tipo de mensaje ({{kind}}). Véalo en el teléfono o en la app del canal.",
        },
        viewOnce: {
          title: "Ver una vez",
          hint: "WhatsApp solo entrega este contenido en el teléfono. Ábralo allí.",
        },
        pix: {
          title: "Pagar con PIX",
          key: "Clave",
          copy: "Copiar clave PIX",
          copied: "Copiado",
        },
        event: {
          fallbackName: "Evento",
          canceled: "Evento cancelado",
          until: "hasta",
          join: "Unirse a la reunión",
        },
        poll: {
          single: "Selecciona una opción",
          multiple: "Selecciona hasta {{limit}} opciones",
          telegramSingle: "Encuesta",
          telegramMultiple: "Encuesta · varias respuestas",
          vote_zero: "{{count}} votos",
          vote_one: "{{count}} voto",
          vote_other: "{{count}} votos",
          send: "Votar",
          sending: "Enviando…",
          sent: "Voto enviado",
          remove: "Quitar voto",
        },
        liveLocation: "Ubicación en tiempo real",
        forwarded: "Reenviado",
        forwardedMany: "Reenviado muchas veces",
        previews: {
          unsupported: "⚠️ Mensaje no compatible",
          location: "Ubicación",
          contact: "Contacto",
          carousel: "Carrusel",
        },
        presence: {
          online: "en línea",
          today: "últ. vez hoy a las {{time}}",
          yesterday: "últ. vez ayer a las {{time}}",
          date: "últ. vez el {{date}} a las {{time}}",
        },
        noTicketMessage: "Selecciona un ticket para empezar a chatear.",
      },
      ticketsManager: {
        buttons: {
          newTicket: "Nuevo",
        },
      },
      ticketsQueueSelect: {
        placeholder: "Linhas",
      },
      tickets: {
        lastMessageKind: {
          image: "📷 Foto",
          video: "📹 Video",
          audio: "🎵 Audio",
          document: "📎 Archivo",
          sticker: "Sticker",
          location: "📍 Ubicación",
          vcard: "👤 Contacto",
          poll: "📊 Encuesta",
          event: "📅 Evento",
          pix: "💠 PIX",
          view_once: "👁 Ver una vez",
          unsupported: "⚠️ Mensaje no compatible",
          call: "📞 Llamada",
          other: "📎 Mensaje",
        },
        noMessages: "Sin mensajes",
        history: {
          empty: "Ningún evento registrado todavía.",
          transfer: "Transferido a la cola #{{queue}}",
          status: "Estado cambiado de \"{{from}}\" a \"{{to}}\"",
          assign: "Asignado al usuario #{{user}}",
          unknownQueue: "desconocida",
          unknownUser: "desconocido",
          fallback: "Acción registrada",
          types: { transfer: "Transferencia", status: "Estado", assign: "Asignación" },
        },
        details: {
          connection: "Conexión",
          title: { contact: "Detalles del contacto", group: "Detalles del grupo" },
          close: "Cerrar panel",
          tabs: { data: "Datos", history: "Historial" },
          selectTicketHistory: "Selecciona un ticket para ver el historial",
          selectTicketDetails: "Selecciona un ticket para ver los detalles",
          sections: { ticket: "Ticket", contact: "Contacto", group: "Grupo", client: "Cliente" },
          fields: { number: "Número", agent: "Agente" },
          groupHint: "El avatar y los detalles aparecen tras recibir mensajes del grupo.",
          groupOfWhatsapp: "Grupo de WhatsApp",
          syncContact: "Actualizar contacto",
          syncRequested: "¡Sincronización solicitada a WhatsApp!",
          syncError: "Error al solicitar la actualización.",
          linkedClient: "Cliente vinculado",
          registerClient: "Registrar cliente",
          pipelines: {
            title: "Pipelines",
            add: "Añadir al pipeline",
            remove: "Quitar del pipeline",
            none: "Ningún pipeline vinculado.",
            dialogTitle: "Añadir al pipeline",
            pipelineLabel: "Pipeline",
            stageLabel: "Etapa",
            selectPipeline: "Selecciona un pipeline",
            selectStage: "Selecciona una etapa",
            fallbackPipeline: "Pipeline",
            fallbackStage: "Etapa",
            cancel: "Cancelar",
            save: "Guardar",
            saving: "Guardando...",
            added: "¡Añadido al pipeline!",
            saveError: "Error al guardar el deal.",
            removeError: "Error al quitar el deal.",
          },
          flows: {
            title: "Flujos",
            start: "Iniciar flujo",
            stop: "Detener flujo",
            none: "Ningún flujo activo.",
            noneAvailable: "Ningún flujo activo disponible.",
            dialogTitle: "Iniciar flujo",
            select: "Selecciona un flujo",
            cancel: "Cancelar",
            startButton: "Iniciar",
            starting: "Iniciando...",
            started: "¡Flujo iniciado!",
            alreadyIn: "El contacto ya está en este flujo.",
            startError: "Error al iniciar el flujo.",
            stopped: "Flujo detenido.",
            stopError: "Error al detener el flujo.",
            status: {
              running: "En ejecución",
              waiting_message: "Esperando respuesta",
              waiting_until: "Esperando tiempo",
              waiting_event: "Esperando evento",
              completed: "Completado",
              aborted: "Abortado",
              expired: "Expirado",
            },
          },
          tags: {
            title: "Etiquetas",
            add: "Añadir etiqueta",
            search: "Buscar o crear etiqueta...",
            noneRegistered: "Ninguna etiqueta registrada",
            noneAdded: "Ninguna etiqueta añadida.",
            create: "Crear etiqueta \"{{name}}\"",
            creating: "Creando...",
            updateError: "Error al actualizar las etiquetas del ticket.",
            createError: "Error al crear la etiqueta.",
          },
        },
        toasts: {
          deleted: "El ticket en el que estabas ha sido borrado.",
        },
        notification: {
          message: "Mensaje de",
        },
        tabs: {
          open: { title: "Bandeja" },
          closed: { title: "Resueltos" },
          search: { title: "Buscar" },
        },
        search: {
          placeholder: "Buscar tickets y mensajes.",
        },
        buttons: {
          showAll: "Todos",
        },
      },
      transferTicketModal: {
        title: "Transferir Ticket",
        fieldLabel: "Escriba para buscar usuarios",
        fieldQueueLabel: "Transferir a la cola",
        fieldConnectionLabel: "Transferir to conexión",
        fieldQueuePlaceholder: "Seleccione una cola",
        fieldConnectionPlaceholder: "Seleccione una conexión",
        noOptions: "No se encontraron usuarios con ese nombre",
        buttons: {
          ok: "Transferir",
          cancel: "Cancelar",
        },
      },
      ticketsList: {
        pendingHeader: "Cola",
        assignedHeader: "Trabajando en",
        noTicketsTitle: "¡Nada acá!",
        connectionTitle: "Conexión que se está utilizando actualmente.",
        noTicketsMessage:
          "No se encontraron tickets con este estado o término de búsqueda",
        buttons: {
          accept: "Acceptar",
        },
      },
      newTicketModal: {
        title: "Crear Ticket",
        fieldLabel: "Escribe para buscar un contacto",
        add: "Añadir",
        buttons: {
          ok: "Guardar",
          cancel: "Cancelar",
        },
      },
      mainDrawer: {
        listItems: {
          dashboard: "Dashboard",
          assistants: "Asistentes de IA",
          pipelines: "Pipelines",
          tickets: "Tickets",
          myActivities: "Mis Actividades",
          activities: "Actividades",
          contacts: "Contactos",
          quickAnswers: "Respuestas rápidas",
          flowBuilder: "Flow Builder",
          clients: "Clientes",
          helpdesk: "Helpdesk",
          whatsappGroupsHub: "Grupos WhatsApp",
          calls: "Llamadas",
          administration: "Administración",
          tags: "Etiquetas",
          groups: "Grupos",
          connections: "Conexiones",
          users: "Usuarios",
          queues: "Colas",
          knowledgeBase: "Base de Conocimiento",
          settings: "Configuración",
          swagger: "Swagger",
        },
        appBar: {
          user: {
            profile: "Perfil",
            logout: "Cerrar Sesión",
          },
        },
      },
      notifications: {
        noTickets: "Sin notificaciones.",
      },
      queues: {
        title: "Linhas",
        table: {
          name: "Nombre",
          color: "Color",
          greeting: "Mensaje de saludo",
          connections: "Conexiones",
          actions: "Comportamiento",
        },
        buttons: {
          add: "Agregar cola",
        },
        confirmationModal: {
          deleteTitle: "Eliminar",
          deleteMessage:
            "¿Estás seguro? ¡Esta acción no se puede revertir! Los tickets en esa cola seguirán existiendo, pero ya no tendrán ninguna cola asignada.",
        },
      },
      queueSelect: {
        inputLabel: "Linhas",
      },
      quickAnswers: {
        title: "Respuestas rápidas",
        table: {
          shortcut: "Atajo",
          message: "Respuesta rápida",
          actions: "Acciones",
        },
        buttons: {
          add: "Agregar respuesta rápida",
        },
        toasts: {
          deleted: "Respuesta rápida eliminada correctamente",
        },
        searchPlaceholder: "Buscar ...",
        confirmationModal: {
          deleteTitle:
            "¿Está seguro de que desea eliminar esta respuesta rápida?",
          deleteMessage: "Esta acción no se puede deshacer.",
        },
      },
      flowBuilder: {
        nodes: {
          quickAnswer: {
            paletteLabel: "Respuesta Rápida",
            title: "Configurar Respuesta Rápida",
            fieldLabel: "Respuesta Rápida",
            placeholder: "Seleccione una respuesta rápida",
            empty: "No hay respuestas rápidas registradas",
            helper: "Envía una plantilla del módulo Respuestas Rápidas.",
          },
        },
      },
      users: {
        title: "Usuarios",
        table: {
          name: "Nombre",
          email: "Correo Electrónico",
          profile: "Perfil",
          whatsapp: "Conexión estándar",
          actions: "Acciones",
        },
        buttons: {
          add: "Agregar usuario",
        },
        toasts: {
          deleted: "Usuario borrado satisfactoriamente.",
        },
        confirmationModal: {
          deleteTitle: "Borrar",
          deleteMessage:
            "Toda la información del usuario se perderá. Los tickets abiertos de los usuarios se moverán a la cola.",
        },
      },
      settings: {
        success: "Configuración guardada satisfactoriamente.",
        title: "Configuración",
        settings: {
          userCreation: {
            name: "Creación de usuarios",
            options: {
              enabled: "Habilitado",
              disabled: "Deshabilitado",
            },
          },
          language: {
            name: "Idioma",
            options: {
              pt: "Portugués",
              en: "Inglés",
              es: "Español",
            },
          },
        },
      },
      messagesList: {
        header: {
          assignedTo: "Asignado a:",
          buttons: {
            return: "Devolver",
            resolve: "Resolver",
            reopen: "Reabrir",
            accept: "Aceptar",
          },
        },
      },
      recoverHistory: {
        button: "Historial",
        title: "Recuperar historial",
        requested: "Recuperación de historial solicitada. Los mensajes aparecerán en breve.",
        ranges: {
          oneDay: "Último día",
          twoDays: "Últimos 2 días",
          oneWeek: "Última semana",
          oneMonth: "Último mes",
          all: "Todo el historial disponible",
        },
      },
      messagesInput: {
        emojiPicker: {
          title: "Emojis",
          search: "Buscar emoji",
          none: "Ningún emoji encontrado",
          loading: "Cargando emojis...",
          categories: {
            "Smileys & Emotion": "Caras y emociones",
            "People & Body": "Personas y cuerpo",
            "Animals & Nature": "Animales y naturaleza",
            "Food & Drink": "Comida y bebida",
            "Travel & Places": "Viajes y lugares",
            Activities: "Actividades",
            Objects: "Objetos",
            Symbols: "Símbolos",
            Flags: "Banderas",
          },
        },
        placeholderOpen: "Escriba un mensaje o presione '' / '' para usar las respuestas rápidas registradas",
        placeholderClosed:
          "Vuelva a abrir o acepte este ticket para enviar un mensaje.",
        signMessage: "Firmar",
      },
      contactDrawer: {
        header: "Detalles del contacto",
        buttons: {
          edit: "Editar contacto",
        },
        extraInfo: "Otra información",
      },
      ticketOptionsMenu: {
        delete: "Borrar",
        transfer: "Transferir",
        confirmationModal: {
          title: "¿Borrar ticket #",
          titleFrom: "del contacto ",
          message:
            "¡Atención! Todos los mensajes Todos los mensajes relacionados con el ticket se perderán.",
        },
        buttons: {
          delete: "Borrar",
          cancel: "Cancelar",
        },
      },
      confirmationModal: {
        buttons: {
          confirm: "Ok",
          cancel: "Cancelar",
        },
      },
      messageOptionsMenu: {
        delete: "Borrar",
        reply: "Responder",
        confirmationModal: {
          title: "¿Borrar mensaje?",
          message: "Esta acción no puede ser revertida.",
        },
      },
      access: {
        title: "Acceso y Permisos",
        metrics: {
          roles: "Funciones activas",
          noRole: "Usuarios sin función",
          total: "Total de usuarios",
        },
        buttons: {
          manageRoles: "Gestionar Funciones",
          manageUsers: "Gestionar Usuarios",
          managePermissions: "Gestionar Permisos",
          legacyGroups: "Grupos heredados",
        },
      },
      backendErrors: {
        ERR_NO_OTHER_WHATSAPP:
          "Debe haber al menos una conexión de WhatsApp predeterminada.",
        ERR_NO_DEF_WAPP_FOUND:
          "No se encontró WhatsApp predeterminado. Verifique la página de conexiones.",
        ERR_WAPP_NOT_INITIALIZED:
          "Esta sesión de WhatsApp no ​​está inicializada. Verifique la página de conexiones.",
        ERR_WAPP_CHECK_CONTACT:
          "No se pudo verificar el contacto de WhatsApp. Verifique la página de conexiones.",
        ERR_WAPP_INVALID_CONTACT: "Este no es un número de whatsapp válido.",
        ERR_WAPP_DOWNLOAD_MEDIA:
          "No se pudieron descargar los medios de WhatsApp. Verifique la página de conexiones.",
        ERR_INVALID_CREDENTIALS: "Error de autenticación. Vuelva a intentarlo.",
        ERR_SENDING_WAPP_MSG:
          "Error al enviar el mensaje de WhatsApp. Verifique la página de conexiones.",
        ERR_DELETE_WAPP_MSG: "No se pudo borrar el mensaje de WhatsApp.",
        ERR_OTHER_OPEN_TICKET: "Ya hay un ticket abierto para este contacto.",
        ERR_SESSION_EXPIRED: "Sesión caducada. Inicie sesión.",
        ERR_USER_CREATION_DISABLED:
          "La creación de usuarios fue deshabilitada por el administrador.",
        ERR_NO_PERMISSION: "No tienes permiso para acceder a este recurso.",
        ERR_DUPLICATED_CONTACT: "Ya existe un contacto con este número.",
        ERR_NO_SETTING_FOUND:
          "No se encontró ninguna configuración con este ID.",
        ERR_NO_CONTACT_FOUND: "No se encontró ningún contacto con este ID.",
        ERR_NO_TICKET_FOUND: "No se encontró ningún ticket con este ID.",
        ERR_NO_USER_FOUND: "No se encontró ningún usuario con este ID.",
        ERR_NO_WAPP_FOUND: "No se encontró WhatsApp con este ID.",
        ERR_CREATING_MESSAGE: "Error al crear el mensaje en la base de datos.",
        ERR_CREATING_TICKET: "Error al crear el ticket en la base de datos.",
        ERR_FETCH_WAPP_MSG:
          "Error al obtener el mensaje en WhtasApp, tal vez sea demasiado antiguo.",
        ERR_QUEUE_COLOR_ALREADY_EXISTS:
          "Este color ya está en uso, elija otro.",
        ERR_WAPP_GREETING_REQUIRED:
          "El mensaje de saludo es obligatorio cuando hay más de una cola.",
        plan_limit_reached: "Se alcanzó el límite de su plan.",
      },
      planLimitResources: {
        users: "usuarios",
        connections: "conexiones",
        queues: "colas",
        plugins: "plugins",
      },
      accountSuspended: {
        suspendedTitle: "Cuenta suspendida",
        suspendedMessage:
          "El acceso a esta cuenta está temporalmente suspendido. Contacte al soporte para regularizar su situación y volver a usar la plataforma.",
        canceledTitle: "Cuenta cancelada",
        canceledMessage:
          "Esta cuenta fue cancelada. Contacte al soporte si cree que esto es un error.",
        backToLogin: "Volver al login",
      },
    },
  },
};

export { messages };
