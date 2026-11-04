/* @jsxImportSource react */
import React from "react";
import { Moon, Sun } from "lucide-react";

import { Button } from "../ui/button";
import { useThemeContext } from "../../context/DarkMode";
import { i18n } from "../../translate/i18n";

const ThemeToggle: React.FC = () => {
  const { darkMode, toggleTheme } = useThemeContext();
  const label = String(i18n.t(darkMode ? "uiText.themeToggle.toLight" : "uiText.themeToggle.toDark"));

  return (
    <Button
      variant="ghost"
      size="icon"
      onClick={toggleTheme}
      aria-label={label}
      title={label}
    >
      {darkMode ? <Sun className="h-5 w-5" /> : <Moon className="h-5 w-5" />}
    </Button>
  );
};

export default ThemeToggle;
